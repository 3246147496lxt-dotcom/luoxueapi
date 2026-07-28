package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type DesktopDiagnostic struct{ ent.Schema }

func (DesktopDiagnostic) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "desktop_diagnostics"}}
}

func (DesktopDiagnostic) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (DesktopDiagnostic) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("device_id"),
		field.Int64("user_id"),
		field.String("public_id").SchemaType(map[string]string{dialect.Postgres: "uuid"}).Unique(),
		field.String("app_version").MaxLen(50),
		field.String("platform").MaxLen(20),
		field.String("architecture").MaxLen(20),
		field.String("os_version").MaxLen(50),
		field.String("gateway_status").MaxLen(20),
		field.String("codex_config_status").MaxLen(20),
		field.Int("request_sample_count").Default(0),
		field.Int("request_error_count").Default(0),
		field.String("encrypted_payload").SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (DesktopDiagnostic) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", DesktopDevice.Type).Ref("diagnostics").Field("device_id").Unique().Required(),
		edge.From("user", User.Type).Ref("desktop_diagnostics").Field("user_id").Unique().Required(),
	}
}

func (DesktopDiagnostic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id", "created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("expires_at"),
	}
}
