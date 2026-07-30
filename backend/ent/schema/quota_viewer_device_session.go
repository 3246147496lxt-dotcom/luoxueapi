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

// QuotaViewerDeviceSession stores only a hash of each rotating refresh token.
type QuotaViewerDeviceSession struct {
	ent.Schema
}

func (QuotaViewerDeviceSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "quota_viewer_device_sessions"}}
}

func (QuotaViewerDeviceSession) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (QuotaViewerDeviceSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("device_id"),
		field.String("family_id").
			SchemaType(map[string]string{dialect.Postgres: "uuid"}),
		field.String("refresh_token_hash").MaxLen(64).Unique(),
		field.String("status").MaxLen(20).Default("active"),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("consumed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("rotation_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "uuid"}),
		field.String("replacement_token_hash").
			MaxLen(64).
			Optional().
			Nillable(),
		field.Time("recovery_expires_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("revoked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (QuotaViewerDeviceSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", QuotaViewerDevice.Type).
			Ref("sessions").
			Field("device_id").
			Unique().
			Required(),
	}
}

func (QuotaViewerDeviceSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id", "status"),
		index.Fields("family_id"),
		index.Fields("expires_at"),
		index.Fields("device_id", "rotation_id").
			Unique().
			StorageKey("idx_quota_viewer_device_sessions_device_rotation").
			Annotations(entsql.IndexWhere("rotation_id IS NOT NULL")),
		index.Fields("recovery_expires_at").
			StorageKey("idx_quota_viewer_device_sessions_recovery_expires").
			Annotations(entsql.IndexWhere("recovery_expires_at IS NOT NULL")),
	}
}
