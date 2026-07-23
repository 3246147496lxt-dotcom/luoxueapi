<template>
        <!-- Tab: Backup -->
        <div v-show="active">
          <BackupSettings />
        </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import BackupSettings from "@/views/admin/BackupView.vue";
import { useSettingsPanelBindings } from "./settingsPanelContext";

export default defineComponent({
  name: "BackupSettingsPanel",
  components: {
    BackupSettings,
  },
  props: {
    active: {
      type: Boolean,
      required: true,
    },
  },
  setup() {
    return useSettingsPanelBindings();
  },
});
</script>
