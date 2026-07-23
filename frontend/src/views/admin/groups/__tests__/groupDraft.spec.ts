import { describe, expect, it } from "vitest";
import {
  createGroupDraft,
  serializeCreateGroupDraft,
  serializeUpdateGroupDraft,
} from "../groupDraft";
import { useGroupEditor } from "../useGroupEditor";

const context = {
  modelRouting: null,
  modelsListConfig: { enabled: false, models: [] },
  supportedModelScopes: [],
};

describe("groupDraft codec", () => {
  it("uses one deterministic default for create and edit", () => {
    expect(createGroupDraft()).toEqual(createGroupDraft());
    expect(createGroupDraft().supported_model_scopes).toEqual([
      "claude",
      "gemini_text",
      "gemini_image",
    ]);
  });

  it("keeps create empty prices unset but encodes edit empty prices as clear", () => {
    const draft = createGroupDraft();
    draft.image_price_1k = "";
    draft.web_search_price_per_call = null;

    expect(serializeCreateGroupDraft(draft, context).image_price_1k).toBeNull();
    expect(
      serializeUpdateGroupDraft(draft, context).image_price_1k,
    ).toBe(-1);
    expect(
      serializeUpdateGroupDraft(draft, context).web_search_price_per_call,
    ).toBe(-1);
  });

  it("normalizes limits and disables invalid batch-image configuration", () => {
    const draft = createGroupDraft();
    draft.daily_limit_usd = "";
    draft.allow_image_generation = true;
    draft.allow_batch_image_generation = true;
    draft.batch_image_discount_multiplier = 0.25;

    const payload = serializeCreateGroupDraft(draft, context);
    expect(payload.daily_limit_usd).toBeNull();
    expect(payload.allow_batch_image_generation).toBe(false);
    expect(payload.batch_image_discount_multiplier).toBe(0.5);
  });

  it("encodes nullable fallback fields with create/update compatibility semantics", () => {
    const draft = createGroupDraft();
    expect(serializeCreateGroupDraft(draft, context).fallback_group_id).toBeNull();
    expect(serializeUpdateGroupDraft(draft, context).fallback_group_id).toBe(0);
  });

  it("tracks one baseline contract for both editor modes", () => {
    const editor = useGroupEditor();
    editor.capture("create");
    expect(editor.isDirty("create")).toBe(false);
    editor.createDraft.name = "changed";
    expect(editor.isDirty("create")).toBe(true);

    editor.reset("edit");
    editor.capture("edit");
    expect(editor.isDirty("edit")).toBe(false);
  });
});
