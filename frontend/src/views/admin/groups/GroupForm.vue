<template>
  <form
    :id="formId"
    @submit.prevent="emit('submit')"
    class="space-y-5"
  >
    <section
      v-if="activeEditorSection === 'general'"
      id="group-editor-section-general"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <div>
      <label :for="nameFieldId" class="input-label">{{ t("admin.groups.form.name") }}</label>
      <input
        :id="nameFieldId"
        v-model="form.name"
        type="text"
        required
        class="input"
        :aria-invalid="editorValidationErrorField === 'name' || undefined"
        :aria-describedby="editorValidationErrorField === 'name' ? nameErrorId : undefined"
        :placeholder="mode === 'create' ? t('admin.groups.enterGroupName') : undefined"
        :data-tour="mode === 'create' ? 'group-form-name' : 'edit-group-form-name'"
        @input="actions.clearValidationError('name')"
      />
      <p
        v-if="editorValidationErrorField === 'name'"
        :id="nameErrorId"
        class="mt-1.5 text-sm text-red-600 dark:text-red-400"
      >
        {{ editorValidationErrorMessage }}
      </p>
    </div>
    <div>
      <label class="input-label">{{
        t("admin.groups.form.description")
      }}</label>
      <textarea
        v-model="form.description"
        rows="3"
        class="input"
        :placeholder="mode === 'create' ? t('admin.groups.optionalDescription') : undefined"
      ></textarea>
    </div>
    <div>
      <label class="input-label">{{
        t("admin.groups.form.platform")
      }}</label>
      <Select
        v-model="form.platform"
        :options="platformOptions"
        :disabled="mode === 'edit'"
        data-tour="group-form-platform"
        @change="handlePlatformChange"
      />
      <p class="input-hint">{{ mode === "create" ? t("admin.groups.platformHint") : t("admin.groups.platformNotEditable") }}</p>
    </div>
    <!-- 从分组复制账号 -->
    <div v-if="copyAccountsGroupOptions.length > 0">
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.copyAccounts.title") }}
        </label>
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="text-xs leading-relaxed text-gray-300">
                {{ mode === "create" ? t("admin.groups.copyAccounts.tooltip") : t("admin.groups.copyAccounts.tooltipEdit") }}
              </p>
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <!-- 已选分组标签 -->
      <div
        v-if="form.copy_accounts_from_group_ids.length > 0"
        class="flex flex-wrap gap-1.5 mb-2"
      >
        <span
          v-for="groupId in form.copy_accounts_from_group_ids"
          :key="groupId"
          class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
        >
          {{
            copyAccountsGroupOptions.find((o) => o.value === groupId)
              ?.label || `#${groupId}`
          }}
          <button
            type="button"
            @click="
              form.copy_accounts_from_group_ids =
                form.copy_accounts_from_group_ids.filter(
                  (id) => id !== groupId,
                )
            "
            class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
          >
            <Icon name="x" size="xs" />
          </button>
        </span>
      </div>
      <!-- 分组选择下拉 -->
      <select
        class="input"
        @change="
          (e) => {
            const val = Number((e.target as HTMLSelectElement).value);
            if (
              val &&
              !form.copy_accounts_from_group_ids.includes(val)
            ) {
              form.copy_accounts_from_group_ids.push(val);
            }
            (e.target as HTMLSelectElement).value = '';
          }
        "
      >
        <option value="">
          {{ t("admin.groups.copyAccounts.selectPlaceholder") }}
        </option>
        <option
          v-for="opt in copyAccountsGroupOptions"
          :key="opt.value"
          :value="opt.value"
          :disabled="
            form.copy_accounts_from_group_ids.includes(opt.value)
          "
        >
          {{ opt.label }}
        </option>
      </select>
      <p class="input-hint">{{ mode === "create" ? t("admin.groups.copyAccounts.hint") : t("admin.groups.copyAccounts.hintEdit") }}</p>
    </div>
    <div>
      <label :for="rateMultiplierFieldId" class="input-label">{{
        t("admin.groups.form.rateMultiplier")
      }}</label>
      <input
        :id="rateMultiplierFieldId"
        v-model.number="form.rate_multiplier"
        type="number"
        step="0.001"
        min="0.001"
        required
        class="input"
        :aria-invalid="editorValidationErrorField === 'rateMultiplier' || undefined"
        :aria-describedby="editorValidationErrorField === 'rateMultiplier' ? rateMultiplierErrorId : undefined"
        data-tour="group-form-multiplier"
        @input="actions.clearValidationError('rateMultiplier')"
      />
      <p
        v-if="editorValidationErrorField === 'rateMultiplier'"
        :id="rateMultiplierErrorId"
        class="mt-1.5 text-sm text-red-600 dark:text-red-400"
      >
        {{ editorValidationErrorMessage }}
      </p>
      <p v-if="mode === 'create'" class="input-hint">{{ t("admin.groups.rateMultiplierHint") }}</p>
    </div>
    <div>
      <label class="input-label">{{ t("admin.groups.form.rpmLimit") }}</label>
      <input
        v-model.number="form.rpm_limit"
        type="number"
        min="0"
        step="1"
        class="input"
        :placeholder="t('admin.groups.form.rpmLimitPlaceholder')"
      />
      <p class="input-hint">{{ t("admin.groups.form.rpmLimitHint") }}</p>
    </div>
    <div
      v-if="form.subscription_type !== 'subscription'"
      :data-tour="mode === 'create' ? 'group-form-exclusive' : undefined"
    >
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.form.exclusive") }}
        </label>
        <!-- Help Tooltip -->
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <!-- Tooltip Popover -->
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="mb-2 text-xs font-medium">
                {{ t("admin.groups.exclusiveTooltip.title") }}
              </p>
              <p class="mb-2 text-xs leading-relaxed text-gray-300">
                {{ t("admin.groups.exclusiveTooltip.description") }}
              </p>
              <div class="rounded bg-gray-800 p-2 dark:bg-gray-700">
                <p class="text-xs leading-relaxed text-gray-300">
                  <span
                    class="inline-flex items-center gap-1 text-primary-400"
                    ><Icon name="lightbulb" size="xs" />
                    {{ t("admin.groups.exclusiveTooltip.example") }}</span
                  >
                  {{ t("admin.groups.exclusiveTooltip.exampleContent") }}
                </p>
              </div>
              <!-- Arrow -->
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <button
          type="button"
          @click="form.is_exclusive = !form.is_exclusive"
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
            form.is_exclusive
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600',
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
              form.is_exclusive ? 'translate-x-6' : 'translate-x-1',
            ]"
          />
        </button>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{
            form.is_exclusive
              ? t("admin.groups.exclusive")
              : t("admin.groups.public")
          }}
        </span>
      </div>
    </div>

    <div v-if="mode === 'edit'">
      <label class="input-label">{{ t("admin.groups.form.status") }}</label>
      <Select v-model="form.status" :options="statusOptions" />
    </div>

    </section>
    <section
      v-if="activeEditorSection === 'access'"
      id="group-editor-section-access"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <!-- Subscription Configuration -->
    <div class="mt-4 border-t pt-4">
      <div>
        <label class="input-label">{{
          t("admin.groups.subscription.type")
        }}</label>
        <Select
          v-model="form.subscription_type"
          :options="subscriptionTypeOptions"
          :disabled="mode === 'edit'"
        />
        <p class="input-hint">
          {{
            mode === "create"
              ? t("admin.groups.subscription.typeHint")
              : t("admin.groups.subscription.typeNotEditable")
          }}
        </p>
      </div>

      <!-- Subscription limits (only show when subscription type is selected) -->
      <div
        v-if="form.subscription_type === 'subscription'"
        class="space-y-4 border-l-2 border-primary-200 pl-4 dark:border-primary-800"
      >
        <div>
          <label class="input-label">{{
            t("admin.groups.subscription.dailyLimit")
          }}</label>
          <input
            v-model.number="form.daily_limit_usd"
            type="number"
            step="0.01"
            min="0"
            class="input"
            :placeholder="t('admin.groups.subscription.noLimit')"
          />
        </div>
        <div>
          <label class="input-label">{{
            t("admin.groups.subscription.weeklyLimit")
          }}</label>
          <input
            v-model.number="form.weekly_limit_usd"
            type="number"
            step="0.01"
            min="0"
            class="input"
            :placeholder="t('admin.groups.subscription.noLimit')"
          />
        </div>
        <div>
          <label class="input-label">{{
            t("admin.groups.subscription.monthlyLimit")
          }}</label>
          <input
            v-model.number="form.monthly_limit_usd"
            type="number"
            step="0.01"
            min="0"
            class="input"
            :placeholder="t('admin.groups.subscription.noLimit')"
          />
        </div>
      </div>
    </div>

    </section>
    <section
      v-if="activeEditorSection === 'models'"
      id="group-editor-section-models"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <div class="border-t pt-4">
      <div class="mb-3 flex items-center justify-between gap-3">
        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t("admin.groups.modelsList.title") }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.groups.modelsList.hint") }}
          </p>
        </div>
        <button
          type="button"
          @click="modelsList.enabled = !modelsList.enabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors',
            modelsList.enabled
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600',
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
              modelsList.enabled ? 'translate-x-6' : 'translate-x-1',
            ]"
          />
        </button>
      </div>
      <div
        v-if="modelsList.enabled"
        class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50/50 dark:border-dark-600 dark:bg-dark-800/40"
      >
        <div
          v-if="!modelsListLoading && modelsList.items.length > 0"
          class="flex items-center justify-between gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 text-xs dark:border-dark-600 dark:bg-dark-800"
        >
          <span class="text-gray-500 dark:text-gray-400">
            {{
              t("admin.groups.modelsList.selectedSummary", {
                selected: modelsListSelectedCount,
                total: modelsList.items.length,
              })
            }}
          </span>
          <div class="flex items-center gap-1.5">
            <button
              type="button"
              class="rounded px-2 py-1 font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
              @click="selectAllModelsListItems(modelsList)"
            >
              {{ t("admin.groups.modelsList.selectAll") }}
            </button>
            <button
              type="button"
              class="rounded px-2 py-1 font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              @click="invertModelsListSelection(modelsList)"
            >
              {{ t("admin.groups.modelsList.invertSelection") }}
            </button>
          </div>
        </div>
        <div
          class="max-h-64 space-y-2 overflow-y-auto p-2"
        >
          <p v-if="modelsListLoading" class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.groups.modelsList.loading") }}
          </p>
          <p
            v-else-if="modelsList.items.length === 0"
            class="text-xs text-gray-500 dark:text-gray-400"
          >
            {{ t("admin.groups.modelsList.empty") }}
          </p>
          <div
            v-for="(item, index) in modelsList.items"
            :key="item.id"
            class="flex items-center gap-2 rounded border border-gray-200 bg-white px-3 py-2 dark:border-dark-600 dark:bg-dark-800"
          >
            <input
              v-model="item.selected"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span class="min-w-0 flex-1 break-all text-sm text-gray-700 dark:text-gray-300">
              {{ item.id }}
            </span>
            <button
              type="button"
              :disabled="index === 0"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-40 dark:hover:bg-dark-600 dark:hover:text-gray-200"
              @click="moveModelItem(index, index - 1)"
            >
              <Icon name="arrowUp" size="sm" />
            </button>
            <button
              type="button"
              :disabled="index === modelsList.items.length - 1"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-40 dark:hover:bg-dark-600 dark:hover:text-gray-200"
              @click="moveModelItem(index, index + 1)"
            >
              <Icon name="arrowDown" size="sm" />
            </button>
          </div>
        </div>
      </div>
    </div>

    </section>
    <section
      v-if="activeEditorSection === 'pricing'"
      id="group-editor-section-pricing"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <!-- 图片生成计费配置 -->
    <div
      v-if="supportsImagePricingPlatform(form.platform)"
      class="border-t pt-4"
    >
      <label
        class="block mb-2 font-medium text-gray-700 dark:text-gray-300"
      >
        {{ t(imagePricingI18nKey(form.platform, "title")) }}
      </label>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
        {{ t(imagePricingI18nKey(form.platform, "description")) }}
      </p>
      <div class="mb-4 grid grid-cols-1 gap-3 md:grid-cols-2">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            v-model="form.allow_image_generation"
            type="checkbox"
            class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          {{ t(imagePricingI18nKey(form.platform, "allowImageGeneration")) }}
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            v-model="form.image_rate_independent"
            type="checkbox"
            class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          {{ t(imagePricingI18nKey(form.platform, "independentMultiplier")) }}
        </label>
      </div>
      <div
        v-if="form.image_rate_independent"
        class="mb-4"
      >
        <label class="input-label">{{
          t(imagePricingI18nKey(form.platform, "imageMultiplier"))
        }}</label>
        <input
          v-model.number="form.image_rate_multiplier"
          type="number"
          step="0.0001"
          min="0"
          class="input"
          placeholder="1"
        />
      </div>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label class="input-label">1K ($)</label>
          <input
            v-model.number="form.image_price_1k"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getImagePricePlaceholder(form.platform, 'image_price_1k')"
          />
        </div>
        <div>
          <label class="input-label">2K ($)</label>
          <input
            v-model.number="form.image_price_2k"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getImagePricePlaceholder(form.platform, 'image_price_2k')"
          />
        </div>
        <div>
          <label class="input-label">4K ($)</label>
          <input
            v-model.number="form.image_price_4k"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getImagePricePlaceholder(form.platform, 'image_price_4k')"
          />
        </div>
      </div>
      <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">
        {{ t(imagePricingI18nKey(form.platform, "modeHint")) }}
      </p>
      <div class="mt-2 rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-gray-800 dark:text-gray-300">
        <div class="mb-1 font-medium">
          {{ t(imagePricingI18nKey(form.platform, "finalPricePreview")) }}
        </div>
        <div class="grid grid-cols-3 gap-2">
          <div
            v-for="item in imageFinalPricePreview"
            :key="item.label"
          >
            {{ item.label }}: {{ item.value }}
          </div>
        </div>
      </div>
      <div v-if="form.platform === 'gemini' && form.allow_image_generation" class="mt-4 border-t border-dashed border-gray-200 pt-4 dark:border-dark-700">
        <label
          class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          <input
            v-model="form.allow_batch_image_generation"
            type="checkbox"
            class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          {{ t("admin.groups.imagePricing.allowBatchImageGeneration") }}
        </label>
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.groups.imagePricing.batchSectionHint") }}
        </p>
        <div
          v-if="form.allow_batch_image_generation"
          class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2"
        >
          <div>
            <label class="input-label">{{
              t("admin.groups.imagePricing.batchDiscountMultiplier")
            }}</label>
            <input
              v-model.number="form.batch_image_discount_multiplier"
              type="number"
              step="0.0001"
              min="0"
              class="input"
              placeholder="0.5"
            />
          </div>
          <div>
            <label class="input-label">{{
              t("admin.groups.imagePricing.batchHoldMultiplier")
            }}</label>
            <input
              v-model.number="form.batch_image_hold_multiplier"
              type="number"
              step="0.0001"
              min="0"
              class="input"
              placeholder="0.6"
            />
          </div>
        </div>
      </div>
      <p
        v-else-if="form.platform !== 'gemini'"
        class="mt-4 border-t border-dashed border-gray-200 pt-4 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400"
      >
        {{ t("admin.groups.imagePricing.batchGeminiOnlyHint") }}
      </p>
    </div>

    <!-- 视频生成计费配置（仅 Grok 平台） -->
    <div
      v-if="supportsVideoPricingPlatform(form.platform)"
      class="border-t pt-4"
    >
      <label
        class="block mb-2 font-medium text-gray-700 dark:text-gray-300"
      >
        {{ t(videoPricingI18nKey("title")) }}
      </label>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
        {{ t(videoPricingI18nKey("description")) }}
      </p>
      <div class="mb-4">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            v-model="form.video_rate_independent"
            type="checkbox"
            class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          {{ t(videoPricingI18nKey("independentMultiplier")) }}
        </label>
      </div>
      <div
        v-if="form.video_rate_independent"
        class="mb-4"
      >
        <label class="input-label">{{
          t(videoPricingI18nKey("videoMultiplier"))
        }}</label>
        <input
          v-model.number="form.video_rate_multiplier"
          type="number"
          step="0.0001"
          min="0"
          class="input"
          placeholder="1"
        />
      </div>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label class="input-label">480p ($/s)</label>
          <input
            v-model.number="form.video_price_480p"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getVideoPricePlaceholder(form.platform, 'video_price_480p')"
          />
        </div>
        <div>
          <label class="input-label">720p ($/s)</label>
          <input
            v-model.number="form.video_price_720p"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getVideoPricePlaceholder(form.platform, 'video_price_720p')"
          />
        </div>
        <div>
          <label class="input-label">1080p ($/s)</label>
          <input
            v-model.number="form.video_price_1080p"
            type="number"
            step="0.001"
            min="0"
            class="input"
            :placeholder="getVideoPricePlaceholder(form.platform, 'video_price_1080p')"
          />
        </div>
      </div>
      <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">
        {{ t(videoPricingI18nKey("modeHint")) }}
      </p>
      <div class="mt-2 rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-gray-800 dark:text-gray-300">
        <div class="mb-1 font-medium">
          {{ t(videoPricingI18nKey("finalPricePreview")) }}
        </div>
        <div class="grid grid-cols-3 gap-2">
          <div
            v-for="item in videoFinalPricePreview"
            :key="item.label"
          >
            {{ item.label }}: {{ item.value }}
          </div>
        </div>
      </div>
    </div>

    <!-- 高峰时段倍率配置（仅订阅类型分组） -->
    <div v-if="form.subscription_type === 'subscription'" class="border-t pt-4">
      <div class="mb-4 grid grid-cols-1 gap-3 md:grid-cols-2">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            v-model="form.peak_rate_enabled"
            type="checkbox"
            class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          <span>{{ t("admin.groups.peakRate.enable") }}</span>
        </label>
      </div>
      <div
        v-if="form.peak_rate_enabled"
        class="mb-4 grid grid-cols-3 gap-3"
      >
        <div>
          <label class="input-label">{{ t("admin.groups.peakRate.peakStart") }}</label>
          <input
            v-model="form.peak_start"
            type="time"
            class="input"
          />
        </div>
        <div>
          <label class="input-label">{{ t("admin.groups.peakRate.peakEnd") }}</label>
          <input
            v-model="form.peak_end"
            type="time"
            class="input"
          />
        </div>
        <div>
          <label class="input-label">{{ t("admin.groups.peakRate.peakMultiplier") }}</label>
          <input
            v-model.number="form.peak_rate_multiplier"
            type="number"
            step="0.001"
            min="0"
            class="input"
            placeholder="1"
            :title="t('admin.groups.peakRate.multiplierHint')"
          />
        </div>
      </div>
    </div>

    </section>
    <section
      v-if="activeEditorSection === 'routing'"
      id="group-editor-section-routing"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <!-- 支持的模型系列（仅 antigravity 平台） -->
    <div v-if="form.platform === 'antigravity'" class="border-t pt-4">
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.supportedScopes.title") }}
        </label>
        <!-- Help Tooltip -->
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="text-xs leading-relaxed text-gray-300">
                {{ t("admin.groups.supportedScopes.tooltip") }}
              </p>
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <div class="space-y-2">
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            :checked="form.supported_model_scopes.includes('claude')"
            @change="toggleScope('claude')"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{
            t("admin.groups.supportedScopes.claude")
          }}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            :checked="
              form.supported_model_scopes.includes('gemini_text')
            "
            @change="toggleScope('gemini_text')"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{
            t("admin.groups.supportedScopes.geminiText")
          }}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            :checked="
              form.supported_model_scopes.includes('gemini_image')
            "
            @change="toggleScope('gemini_image')"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{
            t("admin.groups.supportedScopes.geminiImage")
          }}</span>
        </label>
      </div>
      <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {{ t("admin.groups.supportedScopes.hint") }}
      </p>
    </div>

    <!-- MCP XML 协议注入（仅 antigravity 平台） -->
    <div v-if="form.platform === 'antigravity'" class="border-t pt-4">
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.mcpXml.title") }}
        </label>
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="text-xs leading-relaxed text-gray-300">
                {{ t("admin.groups.mcpXml.tooltip") }}
              </p>
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <button
          type="button"
          @click="form.mcp_xml_inject = !form.mcp_xml_inject"
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
            form.mcp_xml_inject
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600',
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
              form.mcp_xml_inject ? 'translate-x-6' : 'translate-x-1',
            ]"
          />
        </button>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{
            form.mcp_xml_inject
              ? t("admin.groups.mcpXml.enabled")
              : t("admin.groups.mcpXml.disabled")
          }}
        </span>
      </div>
    </div>

    <!-- Claude Code 客户端限制（仅 anthropic 平台） -->
    <div v-if="form.platform === 'anthropic'" class="border-t pt-4">
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.claudeCode.title") }}
        </label>
        <!-- Help Tooltip -->
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="text-xs leading-relaxed text-gray-300">
                {{ t("admin.groups.claudeCode.tooltip") }}
              </p>
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <button
          type="button"
          @click="
            form.claude_code_only = !form.claude_code_only
          "
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
            form.claude_code_only
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600',
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
              form.claude_code_only
                ? 'translate-x-6'
                : 'translate-x-1',
            ]"
          />
        </button>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{
            form.claude_code_only
              ? t("admin.groups.claudeCode.enabled")
              : t("admin.groups.claudeCode.disabled")
          }}
        </span>
      </div>
      <!-- 降级分组选择（仅当启用 claude_code_only 时显示） -->
      <div v-if="form.claude_code_only" class="mt-3">
        <label class="input-label">{{
          t("admin.groups.claudeCode.fallbackGroup")
        }}</label>
        <Select
          v-model="form.fallback_group_id"
          :options="fallbackGroupOptions"
          :placeholder="t('admin.groups.claudeCode.noFallback')"
        />
        <p class="input-hint">
          {{ t("admin.groups.claudeCode.fallbackHint") }}
        </p>
      </div>
    </div>

    <!-- Codex 网页搜索按次计费（仅 openai 平台） -->
    <div
      v-if="form.platform === 'openai'"
      class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4"
    >
      <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
        {{ t("admin.groups.webSearchPricing.title") }}
      </h4>
      <div>
        <label class="input-label">{{
          t("admin.groups.webSearchPricing.pricePerCall")
        }}</label>
        <input
          v-model.number="form.web_search_price_per_call"
          type="number"
          step="0.001"
          min="0"
          placeholder="0.01"
          class="input"
        />
        <p class="input-hint">
          {{ t("admin.groups.webSearchPricing.pricePerCallHint") }}
        </p>
        <div
          class="mt-2 rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
        >
          {{
            t("admin.groups.webSearchPricing.finalPricePreview", {
              price: webSearchFinalPricePreview,
            })
          }}
        </div>
      </div>
    </div>

    <!-- OpenAI Messages 调度配置（仅 openai 平台） -->
    <div
      v-if="form.platform === 'openai'"
      class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4"
    >
      <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
        {{ t("admin.groups.openaiMessages.title") }}
      </h4>

      <!-- 允许 Messages 调度开关 -->
      <div class="flex items-center justify-between">
        <label class="text-sm text-gray-600 dark:text-gray-400">{{
          t("admin.groups.openaiMessages.allowDispatch")
        }}</label>
        <button
          type="button"
          @click="
            form.allow_messages_dispatch =
              !form.allow_messages_dispatch
          "
          class="relative inline-flex h-6 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
          :class="
            form.allow_messages_dispatch
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600'
          "
        >
          <span
            class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
            :class="
              form.allow_messages_dispatch
                ? 'translate-x-6'
                : 'translate-x-1'
            "
          />
        </button>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
        {{ t("admin.groups.openaiMessages.allowDispatchHint") }}
      </p>

      <div v-if="form.allow_messages_dispatch" class="mt-3">
        <div
          class="relative overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800"
        >
          <div
            class="border-b border-gray-100 bg-gray-50/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-700/50"
          >
            <div class="flex items-center gap-2">
              <div class="h-2 w-2 rounded-full bg-blue-500"></div>
              <label
                class="text-sm font-medium text-gray-900 dark:text-white"
                >{{
                  t("admin.groups.openaiMessages.familyMappingTitle")
                }}</label
              >
            </div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.groups.openaiMessages.familyMappingHint") }}
            </p>
          </div>
          <div class="p-4">
            <div class="grid gap-4 md:grid-cols-3">
              <div>
                <label class="input-label">{{
                  t("admin.groups.openaiMessages.opusModel")
                }}</label>
                <input
                  v-model="form.opus_mapped_model"
                  type="text"
                  :placeholder="
                    t('admin.groups.openaiMessages.opusModelPlaceholder')
                  "
                  class="input"
                />
              </div>
              <div>
                <label class="input-label">{{
                  t("admin.groups.openaiMessages.sonnetModel")
                }}</label>
                <input
                  v-model="form.sonnet_mapped_model"
                  type="text"
                  :placeholder="
                    t('admin.groups.openaiMessages.sonnetModelPlaceholder')
                  "
                  class="input"
                />
              </div>
              <div>
                <label class="input-label">{{
                  t("admin.groups.openaiMessages.haikuModel")
                }}</label>
                <input
                  v-model="form.haiku_mapped_model"
                  type="text"
                  :placeholder="
                    t('admin.groups.openaiMessages.haikuModelPlaceholder')
                  "
                  class="input"
                />
              </div>
            </div>
          </div>
        </div>

        <div
          class="mt-5 relative overflow-hidden rounded-xl border border-primary-200 bg-white shadow-sm dark:border-primary-900/50 dark:bg-dark-800"
        >
          <div
            class="border-b border-primary-100 bg-primary-50/80 px-4 py-3 dark:border-primary-900/40 dark:bg-primary-900/20"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="flex items-center gap-2">
                  <div class="h-2 w-2 rounded-full bg-primary-500"></div>
                  <label
                    class="text-sm font-medium text-primary-900 dark:text-primary-100"
                    >{{
                      t("admin.groups.openaiMessages.exactMappingTitle")
                    }}</label
                  >
                </div>
                <p
                  class="mt-1 text-xs text-primary-600/90 dark:text-primary-400/90"
                >
                  {{ t("admin.groups.openaiMessages.exactMappingHint") }}
                </p>
              </div>
            </div>
          </div>

          <div class="p-4 bg-gray-50/30 dark:bg-dark-800/30">
            <div
              v-if="form.exact_model_mappings.length === 0"
              class="flex items-center justify-between gap-3 rounded-xl border-2 border-dashed border-primary-200 bg-white px-5 py-4 text-sm text-primary-700 transition-colors hover:border-primary-300 dark:border-primary-900/40 dark:bg-dark-800 dark:text-primary-300 dark:hover:border-primary-800"
            >
              <span>{{
                t("admin.groups.openaiMessages.noExactMappings")
              }}</span>
              <button
                type="button"
                @click="addMessagesDispatchMapping"
                class="flex items-center gap-1.5 text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              >
                <Icon name="plus" size="sm" />
                {{ t("admin.groups.openaiMessages.addExactMapping") }}
              </button>
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="row in form.exact_model_mappings"
                :key="getMessagesDispatchRowKey(row)"
                class="group relative rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-all hover:border-primary-300 hover:shadow-md dark:border-dark-600 dark:bg-dark-700 dark:hover:border-primary-700"
              >
                <div class="flex items-center gap-4">
                  <div
                    class="grid flex-1 gap-4 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:items-start"
                  >
                    <div>
                      <label class="input-label">{{
                        t("admin.groups.openaiMessages.claudeModel")
                      }}</label>
                      <input
                        v-model="row.claude_model"
                        type="text"
                        :placeholder="
                          t(
                            'admin.groups.openaiMessages.claudeModelPlaceholder',
                          )
                        "
                        class="input bg-gray-50 focus:bg-white dark:bg-dark-800 dark:focus:bg-dark-900"
                      />
                    </div>
                    <div
                      class="hidden md:flex md:justify-center md:pt-7 text-primary-300 dark:text-primary-700"
                    >
                      <Icon
                        name="arrowRight"
                        size="sm"
                        class="transition-transform group-hover:translate-x-1"
                      />
                    </div>
                    <div>
                      <label class="input-label">{{
                        t("admin.groups.openaiMessages.targetModel")
                      }}</label>
                      <input
                        v-model="row.target_model"
                        type="text"
                        :placeholder="
                          t(
                            'admin.groups.openaiMessages.targetModelPlaceholder',
                          )
                        "
                        class="input bg-gray-50 focus:bg-white dark:bg-dark-800 dark:focus:bg-dark-900"
                      />
                    </div>
                  </div>
                  <button
                    type="button"
                    @click="removeMessagesDispatchMapping(row)"
                    class="mt-6 flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    :title="
                      t('admin.groups.openaiMessages.removeExactMapping')
                    "
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </div>

              <button
                type="button"
                @click="addMessagesDispatchMapping"
                class="flex w-full items-center justify-center gap-2 rounded-xl border-2 border-dashed border-gray-300 bg-white py-3 text-sm font-medium text-gray-500 transition-all hover:border-primary-300 hover:bg-primary-50/50 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400 dark:hover:border-primary-800 dark:hover:bg-primary-900/20 dark:hover:text-primary-400"
              >
                <Icon name="plus" size="sm" />
                {{ t("admin.groups.openaiMessages.addExactMapping") }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 账号过滤控制 (OpenAI/Antigravity/Anthropic/Gemini) -->
    <div
      v-if="
        ['openai', 'antigravity', 'anthropic', 'gemini'].includes(
          form.platform,
        )
      "
      class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4 space-y-4"
    >
      <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
        {{ t("admin.groups.accountFilters.title") }}
      </h4>

      <!-- require_oauth_only toggle -->
      <div class="flex items-center justify-between">
        <div>
          <label class="text-sm text-gray-600 dark:text-gray-400"
            >{{ t("admin.groups.accountFilters.oauthOnly") }}</label
          >
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
            {{
              form.require_oauth_only
                ? t("admin.groups.accountFilters.oauthOnlyEnabled")
                : t("admin.groups.accountFilters.disabled")
            }}
          </p>
        </div>
        <button
          type="button"
          @click="
            form.require_oauth_only = !form.require_oauth_only
          "
          class="relative inline-flex h-6 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
          :class="
            form.require_oauth_only
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600'
          "
        >
          <span
            class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
            :class="
              form.require_oauth_only
                ? 'translate-x-6'
                : 'translate-x-1'
            "
          />
        </button>
      </div>

      <!-- require_privacy_set toggle -->
      <div class="flex items-center justify-between">
        <div>
          <label class="text-sm text-gray-600 dark:text-gray-400"
            >{{ t("admin.groups.accountFilters.privacySetOnly") }}</label
          >
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
            {{
              form.require_privacy_set
                ? t("admin.groups.accountFilters.privacySetOnlyEnabled")
                : t("admin.groups.accountFilters.disabled")
            }}
          </p>
        </div>
        <button
          type="button"
          @click="
            form.require_privacy_set = !form.require_privacy_set
          "
          class="relative inline-flex h-6 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
          :class="
            form.require_privacy_set
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600'
          "
        >
          <span
            class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
            :class="
              form.require_privacy_set
                ? 'translate-x-6'
                : 'translate-x-1'
            "
          />
        </button>
      </div>
    </div>

    <!-- 无效请求兜底（仅 anthropic/antigravity 平台，且非订阅分组） -->
    <div
      v-if="
        ['anthropic', 'antigravity'].includes(form.platform) &&
        form.subscription_type !== 'subscription'
      "
      class="border-t pt-4"
    >
      <label class="input-label">{{
        t("admin.groups.invalidRequestFallback.title")
      }}</label>
      <Select
        v-model="form.fallback_group_id_on_invalid_request"
        :options="invalidRequestFallbackOptions"
        :placeholder="t('admin.groups.invalidRequestFallback.noFallback')"
      />
      <p class="input-hint">
        {{ t("admin.groups.invalidRequestFallback.hint") }}
      </p>
    </div>

    </section>
    <section
      v-if="activeEditorSection === 'advanced'"
      id="group-editor-section-advanced"
      class="space-y-5"
      role="region"
      aria-labelledby="group-editor-active-title"
      data-test="group-editor-active-section"
    >
    <!-- 模型路由配置（仅 anthropic 平台） -->
    <div v-if="form.platform === 'anthropic'" class="border-t pt-4">
      <div class="mb-1.5 flex items-center gap-1">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.groups.modelRouting.title") }}
        </label>
        <!-- Help Tooltip -->
        <div class="group relative inline-flex">
          <Icon
            name="questionCircle"
            size="sm"
            :stroke-width="2"
            class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
          />
          <div
            class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-80 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100"
          >
            <div
              class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800"
            >
              <p class="text-xs leading-relaxed text-gray-300">
                {{ t("admin.groups.modelRouting.tooltip") }}
              </p>
              <div
                class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"
              ></div>
            </div>
          </div>
        </div>
      </div>
      <!-- 启用开关 -->
      <div class="flex items-center gap-3 mb-3">
        <button
          type="button"
          @click="
            form.model_routing_enabled =
              !form.model_routing_enabled
          "
          :class="[
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
            form.model_routing_enabled
              ? 'bg-primary-500'
              : 'bg-gray-300 dark:bg-dark-600',
          ]"
        >
          <span
            :class="[
              'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
              form.model_routing_enabled
                ? 'translate-x-6'
                : 'translate-x-1',
            ]"
          />
        </button>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{
            form.model_routing_enabled
              ? t("admin.groups.modelRouting.enabled")
              : t("admin.groups.modelRouting.disabled")
          }}
        </span>
      </div>
      <p
        v-if="!form.model_routing_enabled"
        class="text-xs text-gray-500 dark:text-gray-400 mb-3"
      >
        {{ t("admin.groups.modelRouting.disabledHint") }}
      </p>
      <p v-else class="text-xs text-gray-500 dark:text-gray-400 mb-3">
        {{ t("admin.groups.modelRouting.noRulesHint") }}
      </p>
      <!-- 路由规则列表（仅在启用时显示） -->
      <div v-if="form.model_routing_enabled" class="space-y-3">
        <div
          v-for="rule in routingRules"
          :key="actions.getRuleRenderKey(rule)"
          class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
        >
          <div class="flex items-start gap-3">
            <div class="flex-1 space-y-2">
              <div>
                <label class="input-label text-xs">{{
                  t("admin.groups.modelRouting.modelPattern")
                }}</label>
                <input
                  v-model="rule.pattern"
                  type="text"
                  class="input text-sm"
                  :placeholder="
                    t('admin.groups.modelRouting.modelPatternPlaceholder')
                  "
                />
              </div>
              <div>
                <label class="input-label text-xs">{{
                  t("admin.groups.modelRouting.accounts")
                }}</label>
                <!-- 已选账号标签 -->
                <div
                  v-if="rule.accounts.length > 0"
                  class="flex flex-wrap gap-1.5 mb-2"
                >
                  <span
                    v-for="account in rule.accounts"
                    :key="account.id"
                    class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                  >
                    {{ account.name }}
                    <button
                      type="button"
                      @click="actions.removeSelectedAccount(rule, account.id)"
                      class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
                    >
                      <Icon name="x" size="xs" />
                    </button>
                  </span>
                </div>
                <!-- 账号搜索输入框 -->
                <div class="relative account-search-container">
                  <input
                    v-model="
                      searchKeywords[actions.getRuleSearchKey(rule)]
                    "
                    type="text"
                    class="input text-sm"
                    :placeholder="
                      t(
                        'admin.groups.modelRouting.searchAccountPlaceholder',
                      )
                    "
                    @input="actions.searchAccountsByRule(rule)"
                    @focus="actions.onAccountSearchFocus(rule)"
                  />
                  <!-- 搜索结果下拉框 -->
                  <div
                    v-if="
                      showAccountDropdown[actions.getRuleSearchKey(rule)] &&
                      accountSearchResults[actions.getRuleSearchKey(rule)]
                        ?.length > 0
                    "
                    class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
                  >
                    <button
                      v-for="account in accountSearchResults[
                        actions.getRuleSearchKey(rule)
                      ]"
                      :key="account.id"
                      type="button"
                      @click="actions.selectAccount(rule, account)"
                      class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
                      :class="{
                        'opacity-50': rule.accounts.some(
                          (a) => a.id === account.id,
                        ),
                      }"
                      :disabled="
                        rule.accounts.some((a) => a.id === account.id)
                      "
                    >
                      <span>{{ account.name }}</span>
                      <span class="ml-2 text-xs text-gray-400"
                        >#{{ account.id }}</span
                      >
                    </button>
                  </div>
                </div>
                <p class="text-xs text-gray-400 mt-1">
                  {{ t("admin.groups.modelRouting.accountsHint") }}
                </p>
              </div>
            </div>
            <button
              type="button"
              @click="actions.removeRoutingRule(rule)"
              class="mt-5 p-1.5 text-gray-400 hover:text-red-500 transition-colors"
              :title="t('admin.groups.modelRouting.removeRule')"
            >
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>
      </div>
      <!-- 添加规则按钮（仅在启用时显示） -->
      <button
        v-if="form.model_routing_enabled"
        type="button"
        @click="actions.addRoutingRule"
        class="mt-3 flex items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
      >
        <Icon name="plus" size="sm" />
        {{ t("admin.groups.modelRouting.addRule") }}
      </button>
    </div>
    </section>
  </form>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import Select, { type SelectOption } from "@/components/common/Select.vue";
import Icon from "@/components/icons/Icon.vue";
import { createStableObjectKeyResolver } from "@/utils/stableObjectKey";
import {
  invertModelsListSelection,
  moveModelsListItem,
  selectAllModelsListItems,
  type ModelsListState,
} from "../groupsModelsList";
import type { MessagesDispatchMappingRow } from "../groupsMessagesDispatch";
import {
  getImagePricePlaceholder,
  getVideoPricePlaceholder,
  imagePricingI18nKey,
  supportsImagePricingPlatform,
  supportsVideoPricingPlatform,
  videoPricingI18nKey,
} from "../groupsImagePricing";
import type { GroupDraft } from "./groupDraft";
import type { GroupEditorSectionId } from "./groupEditorRoute";
import type {
  GroupFormActions,
  GroupFormAccount,
  GroupFormMode,
  GroupFormRoutingRule,
} from "./groupForm";

interface GroupStringOption extends SelectOption {
  value: string;
  label: string;
}

interface GroupCopyOption extends SelectOption {
  value: number;
  label: string;
}

interface GroupFallbackOption extends SelectOption {
  value: number | null;
  label: string;
}

interface GroupPricePreview {
  label: string;
  value: string;
}

const props = defineProps<{
  mode: GroupFormMode;
  draft: GroupDraft;
  activeEditorSection: GroupEditorSectionId;
  editorValidationErrorField: "name" | "rateMultiplier" | null;
  editorValidationErrorMessage: string;
  platformOptions: GroupStringOption[];
  statusOptions: GroupStringOption[];
  subscriptionTypeOptions: GroupStringOption[];
  copyAccountsGroupOptions: GroupCopyOption[];
  fallbackGroupOptions: GroupFallbackOption[];
  invalidRequestFallbackOptions: GroupFallbackOption[];
  modelsListState: ModelsListState;
  modelsListLoading: boolean;
  modelsListSelectedCount: number;
  imageFinalPricePreview: GroupPricePreview[];
  videoFinalPricePreview: GroupPricePreview[];
  webSearchFinalPricePreview: string;
  modelRoutingRules: GroupFormRoutingRule[];
  accountSearchKeyword: Record<string, string>;
  accountSearchResults: Record<string, GroupFormAccount[]>;
  showAccountDropdown: Record<string, boolean>;
  actions: GroupFormActions;
}>();

const emit = defineEmits<{
  submit: [];
}>();

const { t } = useI18n();
const form = computed(() => props.draft);
const modelsList = computed(() => props.modelsListState);
const routingRules = computed(() => props.modelRoutingRules);
const searchKeywords = computed(() => props.accountSearchKeyword);
const formId = computed(() => `${props.mode}-group-form`);
const nameFieldId = computed(() => `${props.mode}-group-name`);
const nameErrorId = computed(() => `${props.mode}-group-name-error`);
const rateMultiplierFieldId = computed(
  () => `${props.mode}-group-rate-multiplier`,
);
const rateMultiplierErrorId = computed(
  () => `${props.mode}-group-rate-multiplier-error`,
);
const getMessagesDispatchRowKey =
  createStableObjectKeyResolver<MessagesDispatchMappingRow>(
    "group-form-messages-dispatch-row",
  );

const handlePlatformChange = () => {
  if (props.mode === "create") {
    form.value.copy_accounts_from_group_ids = [];
  }
};

const moveModelItem = (fromIndex: number, toIndex: number) => {
  moveModelsListItem(props.modelsListState, fromIndex, toIndex);
};

const toggleScope = (scope: string) => {
  const index = form.value.supported_model_scopes.indexOf(scope);
  if (index === -1) {
    form.value.supported_model_scopes.push(scope);
  } else {
    form.value.supported_model_scopes.splice(index, 1);
  }
};

const addMessagesDispatchMapping = () => {
  form.value.exact_model_mappings.push({
    claude_model: "",
    target_model: "",
  });
};

const removeMessagesDispatchMapping = (row: MessagesDispatchMappingRow) => {
  const index = form.value.exact_model_mappings.indexOf(row);
  if (index !== -1) {
    form.value.exact_model_mappings.splice(index, 1);
  }
};
</script>
