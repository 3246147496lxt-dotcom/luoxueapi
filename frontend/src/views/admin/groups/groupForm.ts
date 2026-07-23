export type GroupFormMode = "create" | "edit";

export interface GroupFormAccount {
  id: number;
  name: string;
}

export interface GroupFormRoutingRule {
  pattern: string;
  accounts: GroupFormAccount[];
}

export interface GroupFormActions {
  clearValidationError(field?: "name" | "rateMultiplier"): void;
  getRuleRenderKey(rule: GroupFormRoutingRule): string;
  getRuleSearchKey(rule: GroupFormRoutingRule): string;
  searchAccountsByRule(rule: GroupFormRoutingRule): void;
  onAccountSearchFocus(rule: GroupFormRoutingRule): void;
  selectAccount(
    rule: GroupFormRoutingRule,
    account: GroupFormAccount,
  ): void;
  removeSelectedAccount(
    rule: GroupFormRoutingRule,
    accountId: number,
  ): void;
  addRoutingRule(): void;
  removeRoutingRule(rule: GroupFormRoutingRule): void;
}
