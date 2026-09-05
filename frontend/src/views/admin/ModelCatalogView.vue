<template>
  <AppLayout variant="home-clay">
    <TablePageLayout>
      <template #header>
        <AdminPageHeader
          :title="t('admin.modelCatalog.title')"
          :description="t('admin.modelCatalog.description')"
        >
          <template #meta>
            <span>{{ t('admin.modelCatalog.resultCount', { count: total }) }}</span>
          </template>
        </AdminPageHeader>
      </template>

      <template #actions>
        <div
          class="flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <div
                class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                <Icon name="globe" size="md" aria-hidden="true" />
              </div>
              <div>
                <h2 class="text-base font-semibold text-gray-950 dark:text-white">
                  {{ t('admin.modelCatalog.workflowTitle') }}
                </h2>
                <p class="mt-0.5 max-w-3xl text-sm text-gray-600 dark:text-dark-300">
                  {{ t('admin.modelCatalog.workflowDescription') }}
                </p>
              </div>
            </div>
          </div>
          <div class="flex flex-shrink-0 flex-wrap gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadModels"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span class="ml-1.5">{{ t('common.refresh') }}</span>
            </button>
            <button type="button" class="btn btn-primary" @click="openCandidates">
              <Icon name="sparkles" size="sm" />
              <span class="ml-1.5">{{ t('admin.modelCatalog.discoverCandidates') }}</span>
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div class="relative min-w-0 flex-1 sm:max-w-sm">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="searchQuery"
              type="search"
              class="input pl-9"
              :placeholder="t('admin.modelCatalog.searchPlaceholder')"
              @input="scheduleSearch"
              @keydown.enter.prevent="loadModels"
            />
          </div>
          <Select
            v-model="statusFilter"
            :options="statusFilterOptions"
            class="w-full sm:w-44"
            @change="loadModels"
          />
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="models" :loading="loading" row-key="id">
          <template #cell-model="{ row }">
            <div class="flex min-w-[220px] items-center gap-3">
              <div
                class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
              >
                <ModelIcon :model="row.logo_key || row.model" size="20px" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-1.5">
                  <span class="truncate font-medium text-gray-950 dark:text-white">
                    {{ localizedName(row) }}
                  </span>
                  <Icon
                    v-if="row.featured"
                    name="sparkles"
                    size="xs"
                    class="flex-shrink-0 text-amber-500"
                    :title="t('admin.modelCatalog.featured')"
                  />
                </div>
                <code class="mt-0.5 block max-w-[270px] truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ row.model }}
                </code>
              </div>
            </div>
          </template>

          <template #cell-platform="{ row }">
            <div class="space-y-1">
              <span class="badge badge-gray">{{ platformLabel(row.platform) }}</span>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ row.provider || '—' }}</p>
            </div>
          </template>

          <template #cell-public_group_id="{ row }">
            <div class="min-w-[150px]">
              <p class="text-sm font-medium text-gray-800 dark:text-gray-200">
                {{ row.public_group?.name || t('admin.modelCatalog.groupNotSelected') }}
              </p>
              <p v-if="row.public_group" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.modelCatalog.rateMultiplier', { rate: formatMultiplier(row.public_group.rate_multiplier) }) }}
              </p>
            </div>
          </template>

          <template #cell-status="{ row }">
            <div class="space-y-1.5">
              <span :class="['badge', statusClass(row.status)]">
                {{ statusLabel(row.status) }}
              </span>
              <div
                v-if="row.validation && !row.validation.valid"
                class="flex items-center gap-1 text-xs text-amber-700 dark:text-amber-300"
                :title="row.validation.errors.join('\n')"
              >
                <Icon name="exclamationTriangle" size="xs" />
                <span>{{ t('admin.modelCatalog.validationIssues', { count: row.validation.errors.length }) }}</span>
              </div>
            </div>
          </template>

          <template #cell-sort_order="{ row }">
            <span class="tabular-nums text-gray-600 dark:text-dark-300">{{ row.sort_order }}</span>
          </template>

          <template #cell-updated_at="{ row }">
            <span class="whitespace-nowrap text-xs text-gray-500 dark:text-dark-400">
              {{ formatDateTime(row.updated_at) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-1">
              <button
                type="button"
                class="catalog-icon-button hover:bg-blue-50 hover:text-blue-700 dark:hover:bg-blue-900/20 dark:hover:text-blue-300"
                :title="t('admin.modelCatalog.preview')"
                @click="openPreview(row)"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                type="button"
                class="catalog-icon-button hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white"
                :title="t('common.edit')"
                @click="openEditor(row)"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                v-if="row.status !== 'published'"
                type="button"
                class="catalog-icon-button hover:bg-emerald-50 hover:text-emerald-700 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-300"
                :title="t('admin.modelCatalog.publish')"
                :disabled="operatingId === row.id"
                @click="handlePublish(row)"
              >
                <Icon name="upload" size="sm" />
              </button>
              <button
                v-else
                type="button"
                class="catalog-icon-button hover:bg-amber-50 hover:text-amber-700 dark:hover:bg-amber-900/20 dark:hover:text-amber-300"
                :title="t('admin.modelCatalog.unpublish')"
                :disabled="operatingId === row.id"
                @click="requestUnpublish(row)"
              >
                <Icon name="ban" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.modelCatalog.emptyTitle')"
              :description="t('admin.modelCatalog.emptyDescription')"
              :action-text="t('admin.modelCatalog.discoverCandidates')"
              @action="openCandidates"
            />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showCandidatesDialog"
      :title="t('admin.modelCatalog.candidatesTitle')"
      width="extra-wide"
      @close="closeCandidates"
    >
      <div class="space-y-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div class="relative min-w-0 flex-1">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="candidateSearch"
              type="search"
              class="input pl-9"
              :placeholder="t('admin.modelCatalog.candidateSearchPlaceholder')"
            />
          </div>
          <Select
            v-model="candidatePlatform"
            :options="platformFilterOptions"
            class="w-full sm:w-44"
          />
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="candidatesLoading"
            @click="loadCandidates(true)"
          >
            <Icon name="refresh" size="sm" :class="candidatesLoading ? 'animate-spin' : ''" />
            <span class="ml-1.5">{{ t('common.refresh') }}</span>
          </button>
        </div>

        <div
          class="info-surface rounded-xl border px-4 py-3 text-sm"
        >
          {{ t('admin.modelCatalog.candidatesHint') }}
        </div>

        <div v-if="candidatesLoading" class="space-y-2" aria-live="polite">
          <div
            v-for="index in 5"
            :key="index"
            class="h-20 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700"
          ></div>
        </div>
        <div
          v-else-if="filteredCandidates.length === 0"
          class="rounded-xl border border-dashed border-gray-300 px-6 py-12 text-center dark:border-dark-600"
        >
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <p class="mt-3 font-medium text-gray-900 dark:text-white">
            {{ t('admin.modelCatalog.noCandidates') }}
          </p>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.modelCatalog.noCandidatesHint') }}
          </p>
        </div>
        <div v-else class="max-h-[58vh] space-y-2 overflow-y-auto pr-1">
          <article
            v-for="candidate in filteredCandidates"
            :key="candidateKey(candidate)"
            class="flex flex-col gap-3 rounded-xl border border-gray-200 p-4 sm:flex-row sm:items-center dark:border-dark-700"
          >
            <div
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
            >
              <ModelIcon :model="candidate.logo_key || candidate.model" size="22px" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <code class="break-all text-sm font-semibold text-gray-950 dark:text-white">
                  {{ candidate.model }}
                </code>
                <span class="badge badge-gray">{{ platformLabel(candidate.platform) }}</span>
                <span v-if="candidate.provider" class="text-xs text-gray-500 dark:text-dark-400">
                  {{ candidate.provider }}
                </span>
              </div>
              <div class="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('admin.modelCatalog.contextWindow') }}: {{ formatTokenCount(candidate.context_window) }}</span>
                <span>{{ t('admin.modelCatalog.outputLimit') }}: {{ formatTokenCount(candidate.max_output_tokens) }}</span>
                <span>{{ t('admin.modelCatalog.availableOffers', { count: candidate.group_options.length }) }}</span>
              </div>
            </div>
            <div class="flex flex-shrink-0 justify-end">
              <button
                v-if="existingRecord(candidate)"
                type="button"
                class="btn btn-secondary btn-sm"
                @click="openExistingCandidate(candidate)"
              >
                {{ t('admin.modelCatalog.editExisting') }}
              </button>
              <button
                v-else
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="creatingCandidateKey === candidateKey(candidate) || candidate.group_options.length === 0"
                @click="createDraft(candidate)"
              >
                {{
                  creatingCandidateKey === candidateKey(candidate)
                    ? t('common.creating')
                    : t('admin.modelCatalog.createDraft')
                }}
              </button>
            </div>
          </article>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="showEditorDialog"
      :title="t('admin.modelCatalog.editTitle')"
      width="extra-wide"
      @close="closeEditor"
    >
      <form v-if="editor" id="model-catalog-form" class="space-y-6" @submit.prevent="saveEditor">
        <section class="space-y-4">
          <div>
            <h3 class="catalog-section-title">{{ t('admin.modelCatalog.sections.identity') }}</h3>
            <p class="catalog-section-description">{{ t('admin.modelCatalog.sections.identityHint') }}</p>
          </div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.modelId') }}</label>
              <input :value="editor.model" type="text" class="input font-mono" disabled />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.platform') }}</label>
              <input :value="platformLabel(editor.platform)" type="text" class="input" disabled />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.slug') }}</label>
              <input v-model.trim="editor.slug" type="text" class="input font-mono" :placeholder="slugify(editor.platform, editor.model)" />
              <p class="input-hint">{{ t('admin.modelCatalog.slugHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.metadataModelId') }}</label>
              <input v-model.trim="editor.metadata_model_id" type="text" class="input font-mono" :placeholder="editor.model" />
              <p class="input-hint">{{ t('admin.modelCatalog.metadataModelIdHint') }}</p>
            </div>
          </div>
        </section>

        <section class="space-y-4 border-t border-gray-200 pt-5 dark:border-dark-700">
          <div>
            <h3 class="catalog-section-title">{{ t('admin.modelCatalog.sections.content') }}</h3>
            <p class="catalog-section-description">{{ t('admin.modelCatalog.sections.contentHint') }}</p>
          </div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">
                {{ t('admin.modelCatalog.displayNameZh') }}
                <span class="text-red-500">*</span>
              </label>
              <input v-model.trim="editor.display_name_zh" type="text" class="input" maxlength="100" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.displayNameEn') }}</label>
              <input v-model.trim="editor.display_name_en" type="text" class="input" maxlength="100" />
              <p class="input-hint">{{ t('admin.modelCatalog.englishFallbackHint') }}</p>
            </div>
            <div>
              <label class="input-label">
                {{ t('admin.modelCatalog.summaryZh') }}
                <span class="text-red-500">*</span>
              </label>
              <textarea v-model.trim="editor.summary_zh" rows="4" class="input" maxlength="500"></textarea>
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.summaryEn') }}</label>
              <textarea v-model.trim="editor.summary_en" rows="4" class="input" maxlength="500"></textarea>
              <p class="input-hint">{{ t('admin.modelCatalog.englishFallbackHint') }}</p>
            </div>
          </div>
        </section>

        <section class="space-y-4 border-t border-gray-200 pt-5 dark:border-dark-700">
          <div>
            <h3 class="catalog-section-title">{{ t('admin.modelCatalog.sections.metadata') }}</h3>
            <p class="catalog-section-description">{{ t('admin.modelCatalog.sections.metadataHint') }}</p>
          </div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.provider') }}</label>
              <input v-model.trim="editor.provider" type="text" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.logoKey') }}</label>
              <input v-model.trim="editor.logo_key" type="text" class="input font-mono" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.category') }}</label>
              <Select v-model="editor.category" :options="categoryOptions" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.contextWindowOverride') }}</label>
              <input v-model="editor.context_window" type="number" min="1" step="1" class="input" />
              <p class="input-hint">{{ t('admin.modelCatalog.overrideHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.outputLimitOverride') }}</label>
              <input v-model="editor.max_output_tokens" type="number" min="1" step="1" class="input" />
              <p class="input-hint">{{ t('admin.modelCatalog.overrideHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.modelCatalog.sortOrder') }}</label>
              <input v-model="editor.sort_order" type="number" min="0" step="1" class="input" />
            </div>
            <div class="md:col-span-2 lg:col-span-3">
              <label class="input-label">{{ t('admin.modelCatalog.tags') }}</label>
              <input v-model="editor.tags_text" type="text" class="input" :placeholder="t('admin.modelCatalog.tagsPlaceholder')" />
              <p class="input-hint">{{ t('admin.modelCatalog.commaSeparatedHint') }}</p>
            </div>
            <div class="md:col-span-2 lg:col-span-3">
              <label class="input-label">{{ t('admin.modelCatalog.capabilities') }}</label>
              <input v-model="editor.capabilities_text" type="text" class="input" :placeholder="t('admin.modelCatalog.capabilitiesPlaceholder')" />
              <p class="input-hint">{{ t('admin.modelCatalog.capabilitiesHint') }}</p>
            </div>
            <label
              class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-3 md:col-span-2 lg:col-span-3 dark:border-dark-700"
            >
              <span>
                <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.modelCatalog.featured') }}</span>
                <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.featuredHint') }}</span>
              </span>
              <Toggle v-model="editor.featured" />
            </label>
          </div>
        </section>

        <section class="space-y-4 border-t border-gray-200 pt-5 dark:border-dark-700">
          <div>
            <h3 class="catalog-section-title">{{ t('admin.modelCatalog.sections.pricing') }}</h3>
            <p class="catalog-section-description">{{ t('admin.modelCatalog.sections.pricingHint') }}</p>
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.modelCatalog.publicGroup') }}
              <span class="text-red-500">*</span>
            </label>
            <Select
              v-model="editor.public_group_id"
              :options="editorGroupOptions"
              searchable
              clearable
              :placeholder="t('admin.modelCatalog.selectPublicGroup')"
              :empty-text="t('admin.modelCatalog.noPublicGroupOptions')"
            >
              <template #option="{ option }">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-4">
                    <span class="truncate text-sm font-medium">{{ option.label }}</span>
                    <span class="text-xs text-gray-500">{{ option.rateLabel }}</span>
                  </div>
                  <p class="mt-0.5 truncate text-xs text-gray-500 dark:text-dark-400">{{ option.description }}</p>
                </div>
              </template>
            </Select>
            <p class="input-hint">{{ t('admin.modelCatalog.publicGroupHint') }}</p>
          </div>
          <div
            v-if="editorValidationErrors.length"
            class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200"
          >
            <div class="flex items-start gap-2">
              <Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-shrink-0" />
              <div>
                <p class="font-medium">{{ t('admin.modelCatalog.validationTitle') }}</p>
                <ul class="mt-1 list-disc space-y-0.5 pl-4">
                  <li v-for="error in editorValidationErrors" :key="error">{{ error }}</li>
                </ul>
              </div>
            </div>
          </div>
        </section>
      </form>

      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
          <button type="button" class="btn btn-secondary" :disabled="!editor" @click="previewEditor">
            <Icon name="eye" size="sm" />
            <span class="ml-1.5">{{ t('admin.modelCatalog.preview') }}</span>
          </button>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" @click="closeEditor">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" form="model-catalog-form" class="btn btn-primary" :disabled="saving">
              {{ saving ? t('common.saving') : t('admin.modelCatalog.saveDraft') }}
            </button>
          </div>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showPreviewDialog"
      :title="t('admin.modelCatalog.previewTitle')"
      width="wide"
      @close="showPreviewDialog = false"
    >
      <div v-if="previewModel" class="mx-auto max-w-2xl">
        <article class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
          <div class="flex items-start justify-between gap-4">
            <div class="flex min-w-0 items-center gap-3">
              <div
                class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl border border-gray-200 dark:border-dark-700"
              >
                <ModelIcon :model="previewModel.logo_key || previewModel.model" size="26px" />
              </div>
              <div class="min-w-0">
                <h3 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                  {{ localizedName(previewModel) }}
                </h3>
                <code class="mt-0.5 block truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ previewModel.model }}
                </code>
              </div>
            </div>
            <span v-if="previewModel.featured" class="badge badge-warning">
              {{ t('admin.modelCatalog.featured') }}
            </span>
          </div>
          <p class="mt-4 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ localizedSummary(previewModel) || t('admin.modelCatalog.previewMissingSummary') }}
          </p>
          <div class="mt-4 flex flex-wrap gap-2">
            <span class="badge badge-gray">{{ platformLabel(previewModel.platform) }}</span>
            <span v-if="previewModel.category" class="badge badge-gray">{{ categoryLabel(previewModel.category) }}</span>
            <span v-for="capability in previewModel.capabilities" :key="capability" class="badge badge-primary">
              {{ capabilityLabel(capability) }}
            </span>
          </div>
          <dl class="mt-5 grid grid-cols-1 gap-3 border-t border-gray-200 pt-4 sm:grid-cols-3 dark:border-dark-700">
            <div>
              <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.contextWindow') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ formatTokenCount(previewModel.context_window) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.outputLimit') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ formatTokenCount(previewModel.max_output_tokens) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.publicGroup') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                {{ previewModel.public_group?.name || selectedEditorGroupLabel || t('admin.modelCatalog.groupNotSelected') }}
              </dd>
            </div>
          </dl>
          <div class="mt-5 rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.publicPricingPreview') }}</p>
            <div v-if="previewModel.pricing" class="mt-2 grid grid-cols-2 gap-3 text-sm">
              <div>
                <span class="text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.inputPrice') }}</span>
                <p class="mt-0.5 font-semibold text-gray-950 dark:text-white">
                  <CatalogPriceAmount
                    :value="previewModel.pricing.input_price"
                    :currency="previewModel.pricing.currency"
                    :empty-text="t('common.notSet')"
                  />
                  <span v-if="previewModel.pricing.input_price != null && previewModel.pricing.unit">
                    / {{ previewModel.pricing.unit }}
                  </span>
                </p>
              </div>
              <div>
                <span class="text-gray-500 dark:text-dark-400">{{ t('admin.modelCatalog.outputPrice') }}</span>
                <p class="mt-0.5 font-semibold text-gray-950 dark:text-white">
                  <CatalogPriceAmount
                    :value="previewModel.pricing.output_price"
                    :currency="previewModel.pricing.currency"
                    :empty-text="t('common.notSet')"
                  />
                  <span v-if="previewModel.pricing.output_price != null && previewModel.pricing.unit">
                    / {{ previewModel.pricing.unit }}
                  </span>
                </p>
              </div>
            </div>
            <p v-else class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('admin.modelCatalog.pricingResolvedAfterSave') }}
            </p>
          </div>
        </article>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="showUnpublishDialog"
      :title="t('admin.modelCatalog.unpublishTitle')"
      :message="t('admin.modelCatalog.unpublishConfirm', { name: pendingUnpublish ? localizedName(pendingUnpublish) : '' })"
      :confirm-text="t('admin.modelCatalog.unpublish')"
      :cancel-text="t('common.cancel')"
      @confirm="confirmUnpublish"
      @cancel="cancelUnpublish"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  AdminModelCatalogModel,
  CreateModelCatalogRequest,
  ModelCatalogCandidate,
  ModelCatalogStatus,
  UpdateModelCatalogRequest,
} from '@/api/admin/modelCatalog'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import CatalogPriceAmount from '@/components/common/CatalogPriceAmount.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

type CatalogEditor = Omit<AdminModelCatalogModel, 'context_window' | 'max_output_tokens'> & {
  context_window: number | string | null
  max_output_tokens: number | string | null
  tags_text: string
  capabilities_text: string
}

const { t, locale } = useI18n()
const appStore = useAppStore()

const models = ref<AdminModelCatalogModel[]>([])
const total = ref(0)
const loading = ref(false)
const searchQuery = ref('')
const statusFilter = ref<ModelCatalogStatus | 'all'>('all')
const operatingId = ref<number | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const candidates = ref<ModelCatalogCandidate[]>([])
const candidatesLoaded = ref(false)
const candidatesLoading = ref(false)
const showCandidatesDialog = ref(false)
const candidateSearch = ref('')
const candidatePlatform = ref('all')
const creatingCandidateKey = ref('')

const showEditorDialog = ref(false)
const editor = ref<CatalogEditor | null>(null)
const saving = ref(false)
const editorValidationErrors = ref<string[]>([])

const showPreviewDialog = ref(false)
const previewModel = ref<AdminModelCatalogModel | null>(null)

const showUnpublishDialog = ref(false)
const pendingUnpublish = ref<AdminModelCatalogModel | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.modelCatalog.columns.model') },
  { key: 'platform', label: t('admin.modelCatalog.columns.platform') },
  { key: 'public_group_id', label: t('admin.modelCatalog.columns.publicGroup') },
  { key: 'status', label: t('admin.modelCatalog.columns.status') },
  { key: 'sort_order', label: t('admin.modelCatalog.columns.sortOrder') },
  { key: 'updated_at', label: t('admin.modelCatalog.columns.updatedAt') },
  { key: 'actions', label: t('common.actions'), class: 'text-right' },
])

const statusFilterOptions = computed(() => [
  { value: 'all', label: t('admin.modelCatalog.filters.all') },
  { value: 'draft', label: t('admin.modelCatalog.status.draft') },
  { value: 'published', label: t('admin.modelCatalog.status.published') },
  { value: 'archived', label: t('admin.modelCatalog.status.archived') },
])

const platformFilterOptions = computed(() => [
  { value: 'all', label: t('admin.modelCatalog.filters.allPlatforms') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
  { value: 'zhipu', label: 'Zhipu GLM' },
  { value: 'deepseek', label: 'DeepSeek' },
])

const categoryOptions = computed(() => [
  { value: 'chat', label: t('admin.modelCatalog.categories.chat') },
  { value: 'reasoning', label: t('admin.modelCatalog.categories.reasoning') },
  { value: 'code', label: t('admin.modelCatalog.categories.code') },
  { value: 'embedding', label: t('admin.modelCatalog.categories.embedding') },
  { value: 'image', label: t('admin.modelCatalog.categories.image') },
  { value: 'audio', label: t('admin.modelCatalog.categories.audio') },
])

const existingByKey = computed(() => {
  return new Map(models.value.map((item) => [candidateKey(item), item]))
})

const filteredCandidates = computed(() => {
  const query = candidateSearch.value.trim().toLowerCase()
  return candidates.value.filter((candidate) => {
    const platformMatches = candidatePlatform.value === 'all' || candidate.platform === candidatePlatform.value
    const searchMatches = !query || [candidate.model, candidate.platform, candidate.provider, candidate.category]
      .join(' ')
      .toLowerCase()
      .includes(query)
    return platformMatches && searchMatches
  })
})

const currentCandidate = computed(() => {
  if (!editor.value) return null
  const key = candidateKey(editor.value)
  return candidates.value.find((candidate) => candidateKey(candidate) === key) ?? null
})

const editorGroupOptions = computed(() => {
  if (!editor.value) return []
  const options = currentCandidate.value?.group_options ?? []
  const mapped = options.map((option) => ({
    value: option.id,
    label: option.name,
    description: `${option.channel_name} · ${platformLabel(option.platform)}`,
    rateLabel: t('admin.modelCatalog.rateMultiplier', { rate: formatMultiplier(option.rate_multiplier) }),
    pricing: option.pricing,
  }))

  if (
    editor.value.public_group_id &&
    !mapped.some((option) => option.value === editor.value?.public_group_id) &&
    editor.value.public_group
  ) {
    mapped.unshift({
      value: editor.value.public_group.id,
      label: editor.value.public_group.name,
      description: t('admin.modelCatalog.currentGroupUnavailable'),
      rateLabel: t('admin.modelCatalog.rateMultiplier', {
        rate: formatMultiplier(editor.value.public_group.rate_multiplier),
      }),
      pricing: editor.value.pricing ?? emptyPricing(),
    })
  }

  return mapped
})

const selectedEditorGroupLabel = computed(() => {
  if (!editor.value?.public_group_id) return ''
  return editorGroupOptions.value.find((option) => option.value === editor.value?.public_group_id)?.label ?? ''
})

function emptyPricing() {
  return {
    label: '',
    billing_mode: 'token',
    currency: 'CREDIT',
    unit: '',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    peak_rate: { enabled: false, start: null, end: null, multiplier: null },
  }
}

async function loadModels(): Promise<void> {
  loading.value = true
  try {
    const response = await adminAPI.modelCatalog.list({
      status: statusFilter.value,
      search: searchQuery.value.trim(),
    })
    models.value = response.items
    total.value = response.total
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.loadFailed')))
  } finally {
    loading.value = false
  }
}

function scheduleSearch(): void {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => void loadModels(), 280)
}

async function loadCandidates(force = false): Promise<void> {
  if (candidatesLoaded.value && !force) return
  candidatesLoading.value = true
  try {
    const response = await adminAPI.modelCatalog.candidates()
    candidates.value = response.items
    candidatesLoaded.value = true
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.candidatesLoadFailed')))
  } finally {
    candidatesLoading.value = false
  }
}

async function openCandidates(): Promise<void> {
  showCandidatesDialog.value = true
  await loadCandidates()
}

function closeCandidates(): void {
  if (creatingCandidateKey.value) return
  showCandidatesDialog.value = false
}

function candidateKey(candidate: Pick<ModelCatalogCandidate, 'platform' | 'model'>): string {
  return `${candidate.platform}\u0000${candidate.model}`
}

function existingRecord(candidate: ModelCatalogCandidate): AdminModelCatalogModel | null {
  return existingByKey.value.get(candidateKey(candidate)) ?? null
}

async function openExistingCandidate(candidate: ModelCatalogCandidate): Promise<void> {
  const existing = existingRecord(candidate)
  if (!existing) return
  showCandidatesDialog.value = false
  await openEditor(existing)
}

async function createDraft(candidate: ModelCatalogCandidate): Promise<void> {
  const key = candidateKey(candidate)
  creatingCandidateKey.value = key
  try {
    const request: CreateModelCatalogRequest = {
      model: candidate.model,
      platform: candidate.platform,
      metadata_model_id: candidate.model,
      display_name_zh: candidate.model,
      display_name_en: candidate.model,
      summary_zh: '',
      summary_en: '',
      provider: candidate.provider,
      logo_key: candidate.logo_key,
      category: candidate.category || 'chat',
      tags: [],
      capabilities: [],
      context_window: null,
      max_output_tokens: null,
      public_group_id: null,
      featured: false,
      sort_order: 0,
    }
    const created = await adminAPI.modelCatalog.create(request)
    appStore.showSuccess(t('admin.modelCatalog.createSuccess'))
    upsertModel(created)
    showCandidatesDialog.value = false
    await openEditor(created)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.createFailed')))
  } finally {
    creatingCandidateKey.value = ''
  }
}

async function openEditor(model: AdminModelCatalogModel): Promise<void> {
  editorValidationErrors.value = []
  showEditorDialog.value = true
  try {
    await loadCandidates()
    const fresh = await adminAPI.modelCatalog.getById(model.id)
    editor.value = toEditor(fresh)
  } catch (error: unknown) {
    showEditorDialog.value = false
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.loadDetailFailed')))
  }
}

function toEditor(model: AdminModelCatalogModel): CatalogEditor {
  return reactive({
    ...model,
    metadata_model_id: model.metadata_model_id || '',
    display_name_zh: model.display_name_zh || '',
    display_name_en: model.display_name_en || '',
    summary_zh: model.summary_zh || '',
    summary_en: model.summary_en || '',
    provider: model.provider || '',
    logo_key: model.logo_key || '',
    category: model.category || 'chat',
    tags: [...(model.tags || [])],
    capabilities: [...(model.capabilities || [])],
    context_window: model.context_window,
    max_output_tokens: model.max_output_tokens,
    tags_text: (model.tags || []).join(', '),
    capabilities_text: (model.capabilities || []).join(', '),
  }) as CatalogEditor
}

function closeEditor(): void {
  if (saving.value) return
  showEditorDialog.value = false
  editor.value = null
  editorValidationErrors.value = []
}

function buildUpdateRequest(value: CatalogEditor): UpdateModelCatalogRequest {
  return {
    slug: value.slug.trim(),
    model: value.model,
    platform: value.platform,
    metadata_model_id: value.metadata_model_id?.trim() || null,
    display_name_zh: value.display_name_zh.trim(),
    display_name_en: value.display_name_en.trim(),
    summary_zh: value.summary_zh.trim(),
    summary_en: value.summary_en.trim(),
    provider: value.provider.trim(),
    logo_key: value.logo_key.trim(),
    category: value.category,
    tags: splitList(value.tags_text),
    capabilities: splitList(value.capabilities_text),
    context_window: nullablePositiveInteger(value.context_window),
    max_output_tokens: nullablePositiveInteger(value.max_output_tokens),
    public_group_id: Number(value.public_group_id) || null,
    featured: value.featured,
    sort_order: Math.max(0, Number(value.sort_order) || 0),
  }
}

async function saveEditor(): Promise<void> {
  if (!editor.value) return
  saving.value = true
  editorValidationErrors.value = []
  try {
    const updated = await adminAPI.modelCatalog.update(editor.value.id, buildUpdateRequest(editor.value))
    upsertModel(updated)
    editor.value = toEditor(updated)
    appStore.showSuccess(t('admin.modelCatalog.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.saveFailed')))
  } finally {
    saving.value = false
  }
}

function validateForPublish(model: AdminModelCatalogModel): string[] {
  const errors: string[] = []
  if (!model.display_name_zh?.trim()) errors.push(t('admin.modelCatalog.validation.nameZhRequired'))
  if (!model.summary_zh?.trim()) errors.push(t('admin.modelCatalog.validation.summaryZhRequired'))
  if (!model.public_group_id) errors.push(t('admin.modelCatalog.validation.publicGroupRequired'))
  if (model.validation && !model.validation.valid) errors.push(...model.validation.errors)
  return [...new Set(errors)]
}

async function handlePublish(model: AdminModelCatalogModel): Promise<void> {
  const errors = validateForPublish(model)
  if (errors.length > 0) {
    await openEditor(model)
    editorValidationErrors.value = errors
    return
  }
  operatingId.value = model.id
  try {
    const updated = await adminAPI.modelCatalog.publish(model.id)
    upsertModel(updated)
    appStore.showSuccess(t('admin.modelCatalog.publishSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.publishFailed')))
  } finally {
    operatingId.value = null
  }
}

function requestUnpublish(model: AdminModelCatalogModel): void {
  pendingUnpublish.value = model
  showUnpublishDialog.value = true
}

function cancelUnpublish(): void {
  pendingUnpublish.value = null
  showUnpublishDialog.value = false
}

async function confirmUnpublish(): Promise<void> {
  const model = pendingUnpublish.value
  if (!model) return
  showUnpublishDialog.value = false
  operatingId.value = model.id
  try {
    const updated = await adminAPI.modelCatalog.unpublish(model.id)
    upsertModel(updated)
    appStore.showSuccess(t('admin.modelCatalog.unpublishSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.unpublishFailed')))
  } finally {
    operatingId.value = null
    pendingUnpublish.value = null
  }
}

function openPreview(model: AdminModelCatalogModel): void {
  previewModel.value = model
  showPreviewDialog.value = true
}

function previewEditor(): void {
  if (!editor.value) return
  const request = buildUpdateRequest(editor.value)
  const selectedGroup = currentCandidate.value?.group_options.find(
    (option) => option.id === request.public_group_id,
  )
  previewModel.value = {
    ...editor.value,
    ...request,
    tags: request.tags ?? [],
    capabilities: request.capabilities ?? [],
    context_window: request.context_window ?? currentCandidate.value?.context_window ?? null,
    max_output_tokens: request.max_output_tokens ?? currentCandidate.value?.max_output_tokens ?? null,
    public_group: selectedGroup
      ? {
          id: selectedGroup.id,
          name: selectedGroup.name,
          platform: selectedGroup.platform,
          rate_multiplier: selectedGroup.rate_multiplier,
          peak_rate_enabled: selectedGroup.peak_rate_enabled,
          peak_start: selectedGroup.peak_start,
          peak_end: selectedGroup.peak_end,
          peak_rate_multiplier: selectedGroup.peak_rate_multiplier,
        }
      : editor.value.public_group,
    pricing: selectedGroup?.pricing ?? editor.value.pricing,
  } as AdminModelCatalogModel
  showPreviewDialog.value = true
}

function upsertModel(model: AdminModelCatalogModel): void {
  const index = models.value.findIndex((item) => item.id === model.id)
  if (index >= 0) models.value.splice(index, 1, model)
  else models.value.unshift(model)
  total.value = Math.max(total.value, models.value.length)
}

function splitList(value: string): string[] {
  return [...new Set(value.split(/[,，\n]/).map((item) => item.trim()).filter(Boolean))]
}

function nullablePositiveInteger(value: number | string | null): number | null {
  if (value === '' || value === null || value === undefined) return null
  const normalized = Math.floor(Number(value))
  return Number.isFinite(normalized) && normalized > 0 ? normalized : null
}

function slugify(platform: string, model: string): string {
  return `${platform}-${model}`
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function isEnglishLocale(): boolean {
  return String(locale.value).toLowerCase().startsWith('en')
}

function localizedName(model: Pick<AdminModelCatalogModel, 'display_name_zh' | 'display_name_en' | 'model'>): string {
  if (isEnglishLocale()) return model.display_name_en || model.display_name_zh || model.model
  return model.display_name_zh || model.display_name_en || model.model
}

function localizedSummary(model: Pick<AdminModelCatalogModel, 'summary_zh' | 'summary_en'>): string {
  if (isEnglishLocale()) return model.summary_en || model.summary_zh
  return model.summary_zh || model.summary_en
}

function platformLabel(platform: string): string {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    grok: 'Grok',
    zhipu: 'Zhipu GLM',
    deepseek: 'DeepSeek',
  }
  return labels[platform] || platform
}

function statusLabel(status: ModelCatalogStatus): string {
  return t(`admin.modelCatalog.status.${status}`)
}

function statusClass(status: ModelCatalogStatus): string {
  if (status === 'published') return 'badge-success'
  if (status === 'archived') return 'badge-gray'
  return 'badge-warning'
}

function categoryLabel(category: string): string {
  const key = `admin.modelCatalog.categories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}

function capabilityLabel(capability: string): string {
  const key = `admin.modelCatalog.capabilityLabels.${capability}`
  const translated = t(key)
  return translated === key ? capability : translated
}

function formatMultiplier(value: number): string {
  return `${Number(value || 0).toFixed(2).replace(/\.00$/, '')}×`
}

function formatTokenCount(value: number | null | undefined): string {
  if (value === null || value === undefined) return t('common.notSet')
  if (value >= 1_000_000) return `${(value / 1_000_000).toLocaleString(undefined, { maximumFractionDigits: 2 })}M`
  if (value >= 1_000) return `${(value / 1_000).toLocaleString(undefined, { maximumFractionDigits: 1 })}K`
  return value.toLocaleString()
}

onMounted(() => {
  void loadModels()
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style scoped>
.catalog-icon-button {
  @apply inline-flex h-9 w-9 items-center justify-center rounded-lg text-gray-500 transition-colors disabled:cursor-not-allowed disabled:opacity-50 dark:text-dark-300;
}

.catalog-section-title {
  @apply text-sm font-semibold text-gray-950 dark:text-white;
}

.catalog-section-description {
  @apply mt-0.5 text-xs leading-5 text-gray-500 dark:text-dark-400;
}

@media (prefers-reduced-motion: reduce) {
  .catalog-icon-button {
    transition-duration: 1ms;
  }
}
</style>
