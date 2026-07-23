import {
  computed,
  reactive,
  ref,
  type ComputedRef,
  type Ref,
} from "vue";
import {
  cloneGroupDraft,
  createGroupDraft,
  replaceGroupDraft,
  type GroupDraft,
  type GroupDraftMode,
} from "./groupDraft";

type EditableGroupMode = Exclude<GroupDraftMode, never>;

interface UseGroupEditorOptions {
  drafts?: {
    createDraft: GroupDraft;
    editDraft: GroupDraft;
  };
  baseline?: Ref<string>;
  snapshotExtras?: (mode: EditableGroupMode) => unknown;
}

export interface GroupEditorDrafts {
  createDraft: GroupDraft;
  editDraft: GroupDraft;
  baseline: Ref<string>;
  snapshot: (mode: EditableGroupMode) => string;
  capture: (mode: EditableGroupMode) => void;
  reset: (mode: EditableGroupMode, next?: GroupDraft) => void;
  isDirty: (mode: EditableGroupMode) => boolean;
  dirty: (mode: ComputedRef<EditableGroupMode>) => ComputedRef<boolean>;
}

/** Owns create/edit drafts and their shared dirty-state contract. */
export function useGroupEditor(
  options: UseGroupEditorOptions = {},
): GroupEditorDrafts {
  const createDraft =
    options.drafts?.createDraft ??
    (reactive(createGroupDraft()) as GroupDraft);
  const editDraft =
    options.drafts?.editDraft ??
    (reactive(createGroupDraft()) as GroupDraft);
  const baseline = options.baseline ?? ref("");

  const forMode = (mode: EditableGroupMode) =>
    mode === "create" ? createDraft : editDraft;

  const snapshot = (mode: EditableGroupMode) =>
    JSON.stringify({
      draft: cloneGroupDraft(forMode(mode)),
      extras: options.snapshotExtras?.(mode) ?? null,
    });

  const capture = (mode: EditableGroupMode) => {
    baseline.value = snapshot(mode);
  };

  const reset = (mode: EditableGroupMode, next = createGroupDraft()) => {
    replaceGroupDraft(forMode(mode), next);
  };

  const isDirty = (mode: EditableGroupMode) =>
    Boolean(baseline.value) && snapshot(mode) !== baseline.value;

  return {
    createDraft,
    editDraft,
    baseline,
    snapshot,
    capture,
    reset,
    isDirty,
    dirty: (mode) => computed(() => isDirty(mode.value)),
  };
}
