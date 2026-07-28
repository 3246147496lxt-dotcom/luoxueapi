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

// DesktopDevice is an approved macOS installation authorized to use the desktop API.
type DesktopDevice struct {
	ent.Schema
}

func (DesktopDevice) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "desktop_devices"}}
}

func (DesktopDevice) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (DesktopDevice) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("public_id").
			SchemaType(map[string]string{dialect.Postgres: "uuid"}).
			Unique(),
		field.String("installation_id_hash").MaxLen(64),
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("platform").MaxLen(20).Default("macos"),
		field.String("architecture").MaxLen(20),
		field.String("os_version").MaxLen(50).Default(""),
		field.String("app_version").MaxLen(50).Default(""),
		field.String("status").MaxLen(20).Default("pending"),
		field.Int64("token_version").Default(1),
		field.String("release_channel").MaxLen(20).Default("stable"),
		field.Time("pairing_expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("approved_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("activated_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("last_seen_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("revoked_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (DesktopDevice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("desktop_devices").
			Field("user_id").
			Unique().
			Required(),
		edge.To("sessions", DesktopDeviceSession.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("managed_keys", APIKey.Type).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.To("diagnostics", DesktopDiagnostic.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (DesktopDevice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "status"),
		index.Fields("installation_id_hash"),
		index.Fields("pairing_expires_at"),
	}
}
