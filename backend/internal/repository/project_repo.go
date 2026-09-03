package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type projectRepository struct{ db *sql.DB }

func NewProjectRepository(db *sql.DB) service.ProjectRepository { return &projectRepository{db: db} }

const projectCols = `p.public_id,p.name,p.icon,p.color,p.instructions,p.memory_mode,p.created_at,p.updated_at`
const projectReturning = `public_id,name,icon,color,instructions,memory_mode,created_at,updated_at`

func scanProject(scan func(...any) error) (*service.Project, error) {
	var p service.Project
	if err := scan(&p.ID, &p.Name, &p.Icon, &p.Color, &p.Instructions, &p.MemoryMode, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}
func (r *projectRepository) Create(ctx context.Context, userID int64, in service.ProjectInput) (*service.Project, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("project repository db is nil")
	}
	var p service.Project
	err := r.db.QueryRowContext(ctx, `INSERT INTO chat_projects(public_id,user_id,name,icon,color,instructions,memory_mode) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+projectReturning, uuid.NewString(), userID, in.Name, in.Icon, in.Color, in.Instructions, in.MemoryMode).Scan(&p.ID, &p.Name, &p.Icon, &p.Color, &p.Instructions, &p.MemoryMode, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}
func (r *projectRepository) List(ctx context.Context, userID int64) ([]service.Project, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("project repository db is nil")
	}
	rows, e := r.db.QueryContext(ctx, `SELECT `+projectCols+` FROM chat_projects p WHERE p.user_id=$1 AND p.deleted_at IS NULL ORDER BY p.updated_at DESC,p.id DESC`, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []service.Project{}
	for rows.Next() {
		p, e := scanProject(rows.Scan)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := r.hydrateProject(ctx, userID, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (r *projectRepository) Get(ctx context.Context, userID int64, id string) (*service.Project, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("project repository db is nil")
	}
	p, e := scanProject(r.db.QueryRowContext(ctx, `SELECT `+projectCols+` FROM chat_projects p WHERE p.user_id=$1 AND p.public_id=$2 AND p.deleted_at IS NULL`, userID, id).Scan)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, service.ErrProjectNotFound
	}
	if e != nil {
		return nil, e
	}
	if e := r.hydrateProject(ctx, userID, p); e != nil {
		return nil, e
	}
	return p, nil
}

func (r *projectRepository) hydrateProject(ctx context.Context, userID int64, p *service.Project) error {
	if p == nil {
		return errors.New("cannot hydrate a nil project")
	}
	files, err := r.files(ctx, userID, p.ID)
	if err != nil {
		return fmt.Errorf("load project files: %w", err)
	}
	conversations, err := r.conversations(ctx, userID, p.ID)
	if err != nil {
		return fmt.Errorf("load project conversations: %w", err)
	}
	p.Files = files
	p.Conversations = conversations
	return nil
}
func (r *projectRepository) Update(ctx context.Context, userID int64, id string, in service.ProjectInput) (*service.Project, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("project repository db is nil")
	}
	res, e := r.db.ExecContext(ctx, `UPDATE chat_projects SET name=$3,icon=$4,color=$5,instructions=$6,memory_mode=$7,updated_at=NOW() WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL`, userID, id, in.Name, in.Icon, in.Color, in.Instructions, in.MemoryMode)
	if e != nil {
		return nil, e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, service.ErrProjectNotFound
	}
	return r.Get(ctx, userID, id)
}
func (r *projectRepository) Delete(ctx context.Context, userID int64, id string) error {
	if r == nil || r.db == nil {
		return errors.New("project repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var projectID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM chat_projects WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL FOR UPDATE`, userID, id).Scan(&projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrProjectNotFound
		}
		return err
	}
	// Keep the user's conversations and library files intact, but remove the
	// project association just as the UI promises. This also prevents a soft
	// deleted project from retaining hidden context indefinitely.
	if _, err := tx.ExecContext(ctx, `UPDATE chat_conversations SET project_id=NULL,updated_at=NOW() WHERE user_id=$1 AND project_id=$2`, userID, projectID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chat_project_files WHERE user_id=$1 AND project_id=$2`, userID, projectID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE chat_projects SET deleted_at=NOW(),updated_at=NOW() WHERE id=$1 AND user_id=$2`, projectID, userID); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *projectRepository) AddFile(ctx context.Context, userID int64, pid, fid string) (*service.ProjectFile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("project repository db is nil")
	}
	var f service.ProjectFile
	_, e := r.db.ExecContext(ctx, `INSERT INTO chat_project_files(project_id,library_file_id,user_id) SELECT p.id,f.id,$1 FROM chat_projects p JOIN library_files f ON f.public_id=$3 AND f.user_id=$1 AND f.status='ready' WHERE p.public_id=$2 AND p.user_id=$1 AND p.deleted_at IS NULL ON CONFLICT DO NOTHING`, userID, pid, fid)
	if e != nil {
		return nil, e
	}
	e = r.db.QueryRowContext(ctx, `SELECT f.public_id,f.original_name,f.mime_type,f.byte_size,f.created_at FROM chat_project_files pf JOIN chat_projects p ON p.id=pf.project_id JOIN library_files f ON f.id=pf.library_file_id WHERE pf.user_id=$1 AND p.user_id=$1 AND p.public_id=$2 AND p.deleted_at IS NULL AND f.public_id=$3`, userID, pid, fid).Scan(&f.ID, &f.Name, &f.MIMEType, &f.Size, &f.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, service.ErrProjectNotFound
	}
	if e == nil {
		_, _ = r.db.ExecContext(ctx, `UPDATE chat_projects SET updated_at=NOW() WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL`, userID, pid)
	}
	return &f, e
}
func (r *projectRepository) RemoveFile(ctx context.Context, userID int64, pid, fid string) error {
	if r == nil || r.db == nil {
		return errors.New("project repository db is nil")
	}
	res, e := r.db.ExecContext(ctx, `DELETE FROM chat_project_files pf USING chat_projects p,library_files f WHERE pf.project_id=p.id AND pf.library_file_id=f.id AND pf.user_id=$1 AND p.user_id=$1 AND p.public_id=$2 AND p.deleted_at IS NULL AND f.public_id=$3`, userID, pid, fid)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrProjectNotFound
	}
	_, _ = r.db.ExecContext(ctx, `UPDATE chat_projects SET updated_at=NOW() WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL`, userID, pid)
	return nil
}
func (r *projectRepository) MoveConversation(ctx context.Context, userID int64, cid string, pid *string) error {
	if r == nil || r.db == nil {
		return errors.New("project repository db is nil")
	}
	var q string
	var a []any
	if pid == nil {
		q = `UPDATE chat_conversations SET project_id=NULL,updated_at=NOW() WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL`
		a = []any{userID, cid}
	} else {
		q = `UPDATE chat_conversations c SET project_id=p.id,updated_at=NOW() FROM chat_projects p WHERE c.user_id=$1 AND c.public_id=$2 AND c.deleted_at IS NULL AND p.user_id=$1 AND p.public_id=$3 AND p.deleted_at IS NULL`
		a = []any{userID, cid, *pid}
	}
	res, e := r.db.ExecContext(ctx, q, a...)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrProjectNotFound
	}
	if pid != nil {
		_, _ = r.db.ExecContext(ctx, `UPDATE chat_projects SET updated_at=NOW() WHERE user_id=$1 AND public_id=$2 AND deleted_at IS NULL`, userID, *pid)
	}
	return nil
}
func (r *projectRepository) files(ctx context.Context, userID int64, id string) ([]service.ProjectFile, error) {
	rows, e := r.db.QueryContext(ctx, `SELECT f.public_id,f.original_name,f.mime_type,f.byte_size,f.created_at FROM chat_project_files pf JOIN chat_projects p ON p.id=pf.project_id JOIN library_files f ON f.id=pf.library_file_id WHERE pf.user_id=$1 AND p.user_id=$1 AND f.user_id=$1 AND p.public_id=$2 AND p.deleted_at IS NULL AND f.status='ready' ORDER BY pf.created_at DESC`, userID, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []service.ProjectFile{}
	for rows.Next() {
		var f service.ProjectFile
		if e := rows.Scan(&f.ID, &f.Name, &f.MIMEType, &f.Size, &f.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (r *projectRepository) conversations(ctx context.Context, userID int64, id string) ([]service.ChatHistoryConversation, error) {
	rows, e := r.db.QueryContext(ctx, `SELECT c.public_id,c.title,c.model,c.revision,c.version,hm.public_id,c.message_count,c.created_at,c.updated_at,c.deleted_at FROM chat_conversations c JOIN chat_projects p ON p.id=c.project_id LEFT JOIN chat_messages hm ON hm.id=c.head_message_id WHERE c.user_id=$1 AND p.user_id=$1 AND p.public_id=$2 AND p.deleted_at IS NULL AND c.deleted_at IS NULL ORDER BY c.updated_at DESC`, userID, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []service.ChatHistoryConversation{}
	for rows.Next() {
		var c service.ChatHistoryConversation
		var head sql.NullString
		var deleted sql.NullTime
		if e := rows.Scan(&c.ID, &c.Title, &c.Model, &c.Revision, &c.Version, &head, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt, &deleted); e != nil {
			return nil, e
		}
		if head.Valid {
			c.HeadMessageID = &head.String
		}
		if deleted.Valid {
			c.DeletedAt = &deleted.Time
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
