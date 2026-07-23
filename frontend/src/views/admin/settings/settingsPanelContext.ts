import { inject, type InjectionKey } from "vue";
import type { AdminGroup } from "@/types";

export interface DefaultSubscriptionGroupOption {
  value: number;
  label: string;
  description: string | null;
  platform: AdminGroup["platform"];
  subscriptionType: AdminGroup["subscription_type"];
  rate: number;
  [key: string]: unknown;
}

export type SettingsPanelBindings = Record<string, any>;
export type SettingsPanelBindingsProvider = () => SettingsPanelBindings;

export const settingsPanelBindingsKey: InjectionKey<SettingsPanelBindingsProvider> =
  Symbol("settings-panel-bindings");

export function useSettingsPanelBindings(): SettingsPanelBindings {
  const getBindings = inject(settingsPanelBindingsKey);
  if (!getBindings) {
    throw new Error("Settings panels must be rendered inside SettingsView");
  }
  return getBindings();
}
