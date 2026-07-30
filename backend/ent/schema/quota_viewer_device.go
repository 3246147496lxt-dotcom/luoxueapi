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

// QuotaViewerDevice is a macOS or Windows installation authorized only for
// the quota:read audience.
type QuotaViewerDevice struct {
	ent.Schema
}

func (QuotaViewerDevice) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "quota_viewer_devices"}}
}

func (QuotaViewerDevice) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (QuotaViewerDevice) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("public_id").
			SchemaType(map[string]string{dialect.Postgres: "uuid"}).
			Unique(),
		field.String("client_id").MaxLen(64),
		field.String("scope").MaxLen(64),
		field.String("installation_id_hash").MaxLen(64),
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("platform").MaxLen(20),
		field.String("architecture").MaxLen(20),
		field.String("os_version").MaxLen(50).Default(""),
		field.String("app_version").MaxLen(50).Default(""),
		field.String("status").MaxLen(20).Default("pending"),
		field.Int64("token_version").Default(1),
		field.Time("pairing_expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("approved_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("activated_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("last_seen_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("revoked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (QuotaViewerDevice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("quota_viewer_devices").
			Field("user_id").
			Unique().
			Required(),
		edge.To("sessions", QuotaViewerDeviceSession.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (QuotaViewerDevice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status"),
		index.Fields("installation_id_hash"),
		index.Fields("pairing_expires_at"),
	}
}
