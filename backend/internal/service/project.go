package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ProjectMemoryDefault   = "default"
	ProjectMemoryOnly      = "project_only"
	maxProjectName         = 200
	maxProjectInstructions = 32 << 10
)

var (
	ErrProjectInvalid     = infraerrors.BadRequest("PROJECT_INVALID", "The project request is invalid")
	ErrProjectNotFound    = infraerrors.NotFound("PROJECT_NOT_FOUND", "Project not found")
	ErrProjectUnavailable = infraerrors.ServiceUnavailable("PROJECT_UNAVAILABLE", "Projects are temporarily unavailable")
)

type Project struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Icon          string                    `json:"icon"`
	Color         string                    `json:"color"`
	Instructions  string                    `json:"instructions"`
	MemoryMode    string                    `json:"memory_mode"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	Files         []ProjectFile             `json:"files,omitempty"`
	Conversations []ChatHistoryConversation `json:"conversations,omitempty"`
}

type ProjectFile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MIMEType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectInput struct{ Name, Icon, Color, Instructions, MemoryMode string }

type ProjectRepository interface {
	Create(ctx context.Context, userID int64, input ProjectInput) (*Project, error)
	List(ctx context.Context, userID int64) ([]Project, error)
	Get(ctx context.Context, userID int64, id string) (*Project, error)
	Update(ctx context.Context, userID int64, id string, input ProjectInput) (*Project, error)
	Delete(ctx context.Context, userID int64, id string) error
	AddFile(ctx context.Context, userID int64, projectID, fileID string) (*ProjectFile, error)
	RemoveFile(ctx context.Context, userID int64, projectID, fileID string) error
	MoveConversation(ctx context.Context, userID int64, conversationID string, projectID *string) error
}

type ProjectService struct{ repo ProjectRepository }

func NewProjectService(repo ProjectRepository) *ProjectService { return &ProjectService{repo: repo} }

func normalizeProjectInput(input ProjectInput) (ProjectInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Icon = strings.TrimSpace(input.Icon)
	input.Color = strings.TrimSpace(input.Color)
	input.Instructions = strings.TrimSpace(input.Instructions)
	input.MemoryMode = strings.TrimSpace(input.MemoryMode)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > maxProjectName ||
		utf8.RuneCountInString(input.Icon) > 8 || len(input.Instructions) > maxProjectInstructions {
		return ProjectInput{}, ErrProjectInvalid
	}
	if input.Icon == "" {
		input.Icon = "folder"
	}
	if input.Color == "" {
		input.Color = "gray"
	}
	if !validProjectColor(input.Color) {
		return ProjectInput{}, ErrProjectInvalid
	}
	if input.MemoryMode == "" {
		input.MemoryMode = ProjectMemoryDefault
	}
	if input.MemoryMode != ProjectMemoryDefault && input.MemoryMode != ProjectMemoryOnly {
		return ProjectInput{}, ErrProjectInvalid
	}
	return input, nil
}

func validProjectColor(value string) bool {
	if strings.HasPrefix(value, "#") {
		if len(value) != 4 && len(value) != 7 && len(value) != 9 {
			return false
		}
		for _, r := range value[1:] {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
		return true
	}
	switch value {
	case "gray", "red", "orange", "yellow", "green", "blue", "purple", "pink":
		return true
	default:
		return false
	}
}

func (s *ProjectService) Create(ctx context.Context, userID int64, input ProjectInput) (*Project, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrProjectUnavailable
	}
	n, err := normalizeProjectInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, userID, n)
}
func (s *ProjectService) List(ctx context.Context, userID int64) ([]Project, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrProjectUnavailable
	}
	return s.repo.List(ctx, userID)
}
func (s *ProjectService) Get(ctx context.Context, userID int64, id string) (*Project, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrProjectUnavailable
	}
	if strings.TrimSpace(id) == "" {
		return nil, ErrProjectInvalid
	}
	return s.repo.Get(ctx, userID, strings.TrimSpace(id))
}
func (s *ProjectService) Update(ctx context.Context, userID int64, id string, input ProjectInput) (*Project, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrProjectUnavailable
	}
	if strings.TrimSpace(id) == "" {
		return nil, ErrProjectInvalid
	}
	n, err := normalizeProjectInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, strings.TrimSpace(id), n)
}
func (s *ProjectService) Delete(ctx context.Context, userID int64, id string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrProjectUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrProjectInvalid
	}
	return s.repo.Delete(ctx, userID, id)
}
func (s *ProjectService) AddFile(ctx context.Context, userID int64, projectID, fileID string) (*ProjectFile, error) {
	if s == nil || s.repo == nil || userID <= 0 || strings.TrimSpace(projectID) == "" || strings.TrimSpace(fileID) == "" {
		return nil, ErrProjectInvalid
	}
	return s.repo.AddFile(ctx, userID, strings.TrimSpace(projectID), strings.TrimSpace(fileID))
}
func (s *ProjectService) RemoveFile(ctx context.Context, userID int64, projectID, fileID string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrProjectUnavailable
	}
	projectID = strings.TrimSpace(projectID)
	fileID = strings.TrimSpace(fileID)
	if projectID == "" || fileID == "" {
		return ErrProjectInvalid
	}
	return s.repo.RemoveFile(ctx, userID, projectID, fileID)
}
func (s *ProjectService) MoveConversation(ctx context.Context, userID int64, conversationID string, projectID *string) error {
	if s == nil || s.repo == nil || userID <= 0 || strings.TrimSpace(conversationID) == "" {
		return ErrProjectInvalid
	}
	if projectID != nil {
		v := strings.TrimSpace(*projectID)
		if v == "" {
			projectID = nil
		} else {
			projectID = &v
		}
	}
	return s.repo.MoveConversation(ctx, userID, strings.TrimSpace(conversationID), projectID)
}
