import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

import { describe, expect, it } from "vitest";

const currentDir = dirname(fileURLToPath(import.meta.url));
const groupsViewSource = readFileSync(
  resolve(currentDir, "../GroupsView.vue"),
  "utf8",
);
const groupFormSource = readFileSync(
  resolve(currentDir, "../groups/GroupForm.vue"),
  "utf8",
);

describe("groups models list layout", () => {
  it("uses one typed form component for create and edit workspaces", () => {
    expect(groupsViewSource.match(/<GroupForm\b/g)).toHaveLength(2);
    expect(groupFormSource.match(/<form\b/g)).toHaveLength(1);
    expect(groupsViewSource).not.toContain('id="create-group-form"');
    expect(groupsViewSource).not.toContain('id="edit-group-form"');
  });

  it("keeps the toolbar outside of the scrolling list content", () => {
    expect(groupFormSource).toContain("overflow-hidden rounded-lg border");
    expect(groupFormSource).toContain("max-h-64 space-y-2 overflow-y-auto p-2");
    expect(groupFormSource).not.toContain("sticky top-0");
  });
});
