package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type projectRepoStub struct{ input ProjectInput }

func (s *projectRepoStub) Create(_ context.Context, _ int64, in ProjectInput) (*Project, error) {
	s.input = in
	return &Project{ID: "p1", Name: in.Name}, nil
}
func (*projectRepoStub) List(context.Context, int64) ([]Project, error)       { return nil, nil }
func (*projectRepoStub) Get(context.Context, int64, string) (*Project, error) { return nil, nil }
func (*projectRepoStub) Update(context.Context, int64, string, ProjectInput) (*Project, error) {
	return nil, nil
}
func (*projectRepoStub) Delete(context.Context, int64, string) error { return nil }
func (*projectRepoStub) AddFile(context.Context, int64, string, string) (*ProjectFile, error) {
	return nil, nil
}
func (*projectRepoStub) RemoveFile(context.Context, int64, string, string) error        { return nil }
func (*projectRepoStub) MoveConversation(context.Context, int64, string, *string) error { return nil }

func TestProjectServiceNormalizesDefaults(t *testing.T) {
	r := &projectRepoStub{}
	p, err := NewProjectService(r).Create(context.Background(), 7, ProjectInput{Name: "  Demo  "})
	require.NoError(t, err)
	require.Equal(t, "p1", p.ID)
	require.Equal(t, "Demo", r.input.Name)
	require.Equal(t, "folder", r.input.Icon)
	require.Equal(t, "gray", r.input.Color)
	require.Equal(t, ProjectMemoryDefault, r.input.MemoryMode)
}
func TestProjectServiceRejectsInvalidMemoryMode(t *testing.T) {
	_, err := NewProjectService(&projectRepoStub{}).Create(context.Background(), 7, ProjectInput{Name: "x", MemoryMode: "shared"})
	require.ErrorIs(t, err, ErrProjectInvalid)
}

func TestProjectServiceRejectsInvalidColor(t *testing.T) {
	_, err := NewProjectService(&projectRepoStub{}).Create(context.Background(), 7, ProjectInput{Name: "x", Color: "url(javascript:alert(1))"})
	require.ErrorIs(t, err, ErrProjectInvalid)
}
