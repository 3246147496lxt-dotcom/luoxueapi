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

// DesktopDeviceSession stores a hashed rotating refresh-token member.
type DesktopDeviceSession struct {
	ent.Schema
}

func (DesktopDeviceSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "desktop_device_sessions"}}
}

func (DesktopDeviceSession) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (DesktopDeviceSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("device_id"),
		field.String("family_id").SchemaType(map[string]string{dialect.Postgres: "uuid"}),
		field.String("refresh_token_hash").MaxLen(64).Unique(),
		field.String("status").MaxLen(20).Default("active"),
		field.Time("expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("consumed_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("revoked_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (DesktopDeviceSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", DesktopDevice.Type).
			Ref("sessions").
			Field("device_id").
			Unique().
			Required(),
	}
}

func (DesktopDeviceSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id", "status"),
		index.Fields("family_id"),
		index.Fields("expires_at"),
	}
}
