package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ModelCatalogModel is an explicitly curated, public model-catalog entry.
// Pricing is deliberately not copied into this table: public prices are always
// resolved from the selected standard group and its active channel at read time.
type ModelCatalogModel struct {
	ent.Schema
}

func (ModelCatalogModel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "model_catalog_models"},
	}
}

func (ModelCatalogModel) Fields() []ent.Field {
	return []ent.Field{
		field.String("slug").MaxLen(160).NotEmpty().Unique(),
		field.String("model").MaxLen(255).NotEmpty(),
		field.String("platform").MaxLen(64).NotEmpty(),
		field.String("metadata_model_id").MaxLen(255).Default(""),
		field.String("display_name_zh").MaxLen(160).Default(""),
		field.String("display_name_en").MaxLen(160).Default(""),
		field.String("summary_zh").SchemaType(map[string]string{dialect.Postgres: "text"}).Default(""),
		field.String("summary_en").SchemaType(map[string]string{dialect.Postgres: "text"}).Default(""),
		field.String("provider").MaxLen(80).Default(""),
		field.String("logo_key").MaxLen(80).Default(""),
		field.String("category").MaxLen(80).Default(""),
		field.JSON("tags", []string{}).Default([]string{}),
		field.JSON("capabilities", []string{}).Default([]string{}),
		field.Int64("context_window").Optional().Nillable(),
		field.Int64("max_output_tokens").Optional().Nillable(),
		field.Int64("public_group_id").Optional().Nillable(),
		field.String("status").MaxLen(20).Default(domain.ModelCatalogStatusDraft),
		field.Bool("featured").Default(false),
		field.Int("sort_order").Default(0),
		field.Time("published_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ModelCatalogModel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform", "model").Unique(),
		index.Fields("status", "featured", "sort_order", "id"),
		index.Fields("public_group_id"),
	}
}
