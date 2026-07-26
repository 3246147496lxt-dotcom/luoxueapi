<template>
  <AppLayout variant="home-clay">
    <div class="space-y-5 pb-8" data-admin-page-kind="form">
      <section class="rounded-2xl border border-gray-200 bg-white px-5 py-4 dark:border-dark-700 dark:bg-dark-800 sm:px-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="book" size="md" aria-hidden="true" />
              </div>
              <div>
                <h1 class="text-xl font-semibold text-gray-950 dark:text-white">
                  {{ t('admin.documentation.title') }}
                </h1>
                <p class="mt-0.5 max-w-3xl text-sm text-gray-600 dark:text-dark-300">
                  {{ t('admin.documentation.description') }}
                </p>
              </div>
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-2 text-xs">
              <span
                class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 font-medium"
                :class="isDirty
                  ? 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-200'
                  : 'bg-emerald-50 text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-200'"
              >
                <span class="h-1.5 w-1.5 rounded-full" :class="isDirty ? 'bg-amber-500' : 'bg-emerald-500'"></span>
                {{ isDirty ? t('admin.documentation.unsavedChanges') : t('admin.documentation.draftSaved') }}
              </span>
              <span v-if="draftUpdatedAt" class="text-gray-500 dark:text-dark-400">
                {{ t('admin.documentation.lastSaved', { time: formatTimestamp(draftUpdatedAt) }) }}
              </span>
              <span v-if="publishedSnapshot" class="text-gray-500 dark:text-dark-400">
                {{ t('admin.documentation.publishedStatus', { version: publishedSnapshot.version ?? '—', time: formatTimestamp(publishedSnapshot.published_at) }) }}
              </span>
              <span v-else class="text-gray-500 dark:text-dark-400">
                {{ t('admin.documentation.neverPublished') }}
              </span>
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadDocumentation">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" aria-hidden="true" />
              <span class="ml-1.5">{{ t('common.refresh') }}</span>
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="openHistory">
              <Icon name="clock" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ t('admin.documentation.history') }}</span>
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="showPreview = true">
              <Icon name="eye" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ t('admin.documentation.preview') }}</span>
            </button>
            <button type="button" class="btn btn-secondary" :disabled="saving || loading || !isDirty" @click="saveDraft">
              <Icon name="document" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ saving ? t('common.saving') : t('admin.documentation.saveDraft') }}</span>
            </button>
            <button type="button" class="btn btn-primary" :disabled="publishing || loading" @click="requestPublish">
              <Icon name="upload" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ publishing ? t('admin.documentation.publishing') : t('admin.documentation.publish') }}</span>
            </button>
          </div>
        </div>
      </section>

      <div
        v-if="validationErrors.length"
        ref="validationAlert"
        class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-amber-950 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-100"
        role="alert"
        tabindex="-1"
      >
        <div class="flex gap-2.5">
          <Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-shrink-0" aria-hidden="true" />
          <div>
            <p class="text-sm font-semibold">{{ t('admin.documentation.validationTitle') }}</p>
            <ul class="mt-1 list-disc space-y-0.5 pl-5 text-sm">
              <li v-for="error in validationErrors" :key="error">{{ error }}</li>
            </ul>
          </div>
        </div>
      </div>

      <section v-if="loading" class="grid gap-5 lg:grid-cols-[240px_minmax(0,1fr)]" aria-live="polite">
        <div class="h-80 animate-pulse rounded-2xl bg-white dark:bg-dark-800"></div>
        <div class="space-y-4 rounded-2xl bg-white p-6 dark:bg-dark-800">
          <div class="h-8 w-56 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
          <div class="h-11 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
          <div class="h-28 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
          <div class="h-64 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
        </div>
      </section>

      <section
        v-else-if="loadError"
        class="rounded-2xl border border-gray-200 bg-white px-6 py-16 text-center dark:border-dark-700 dark:bg-dark-800"
      >
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" aria-hidden="true" />
        <h2 class="mt-3 text-base font-semibold text-gray-950 dark:text-white">
          {{ t('admin.documentation.loadFailed') }}
        </h2>
        <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ loadError }}</p>
        <button type="button" class="btn btn-primary mt-5" @click="loadDocumentation">
          {{ t('admin.documentation.retry') }}
        </button>
      </section>

      <section v-else class="grid min-w-0 gap-5 lg:grid-cols-[240px_minmax(0,1fr)]">
        <aside class="documentation-category-sidebar self-start rounded-2xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between px-2 pb-2">
            <div>
              <h2 class="text-sm font-semibold text-gray-950 dark:text-white">
                {{ t('admin.documentation.categories') }}
              </h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.documentation.categoryCount', { count: content.tutorials.length }) }}
              </p>
            </div>
            <button
              type="button"
              class="rounded-lg p-2 text-primary-700 hover:bg-primary-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-primary-300 dark:hover:bg-primary-900/20"
              :title="t('admin.documentation.addCategory')"
              @click="addCategory"
            >
              <Icon name="plus" size="sm" aria-hidden="true" />
            </button>
          </div>

          <div v-if="content.tutorials.length" class="space-y-1" role="list">
            <button
              v-for="tutorial in content.tutorials"
              :key="tutorial.id"
              type="button"
              class="group flex w-full items-center gap-2 rounded-xl px-2.5 py-2.5 text-left transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
              :class="activeTutorialId === tutorial.id
                ? 'bg-primary-50 text-primary-950 dark:bg-primary-900/20 dark:text-primary-100'
                : 'text-gray-700 hover:bg-gray-100 dark:text-dark-200 dark:hover:bg-dark-700'"
              role="listitem"
              @click="activeTutorialId = tutorial.id"
            >
              <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-white text-gray-600 dark:bg-dark-800 dark:text-dark-200">
                <DocumentationCategoryIcon
                  :icon="tutorial.icon"
                  :icon-svg="tutorial.icon_svg"
                />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-medium">
                  {{ tutorial.tab_label || t('admin.documentation.untitledCategory') }}
                </span>
                <span class="block text-xs opacity-70">
                  {{ t('admin.documentation.stepCount', { count: tutorial.steps.length }) }}
                </span>
              </span>
            </button>
          </div>

          <div v-else class="rounded-xl border border-dashed border-gray-300 px-3 py-7 text-center dark:border-dark-600">
            <Icon name="book" size="lg" class="mx-auto text-gray-400" aria-hidden="true" />
            <p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.documentation.noCategories') }}
            </p>
            <button type="button" class="mt-3 text-sm font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300" @click="addCategory">
              {{ t('admin.documentation.createFirstCategory') }}
            </button>
          </div>

          <p class="mt-3 border-t border-gray-200 px-2 pt-3 text-xs leading-5 text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('admin.documentation.orderHint') }}
          </p>
        </aside>

        <div v-if="activeTutorial" class="min-w-0 rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-dark-700 sm:px-6">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-950 dark:text-white">
                  {{ activeTutorial.tab_label || t('admin.documentation.untitledCategory') }}
                </h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('admin.documentation.categoryEditorHint') }}
                </p>
              </div>
              <div class="flex items-center gap-1">
                <button type="button" class="editor-icon-button" :disabled="activeTutorialIndex === 0" :title="t('admin.documentation.moveUp')" @click="moveCategory(-1)">
                  <Icon name="arrowUp" size="sm" aria-hidden="true" />
                </button>
                <button type="button" class="editor-icon-button" :disabled="activeTutorialIndex === content.tutorials.length - 1" :title="t('admin.documentation.moveDown')" @click="moveCategory(1)">
                  <Icon name="arrowDown" size="sm" aria-hidden="true" />
                </button>
                <button type="button" class="editor-icon-button text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20" :title="t('admin.documentation.deleteCategory')" @click="requestDeleteCategory">
                  <Icon name="trash" size="sm" aria-hidden="true" />
                </button>
              </div>
            </div>
          </div>

          <div class="space-y-7 px-5 py-5 sm:px-6 sm:py-6">
            <fieldset class="space-y-4">
              <legend class="text-sm font-semibold text-gray-950 dark:text-white">
                {{ t('admin.documentation.categoryDetails') }}
              </legend>
              <div class="grid gap-4 lg:grid-cols-[minmax(280px,0.8fr)_minmax(0,1fr)]">
                <div>
                  <span class="input-label">
                    {{ t('admin.documentation.icon') }}
                  </span>
                  <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
                    <div class="flex items-center gap-3">
                      <span class="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-100">
                        <DocumentationCategoryIcon
                          :icon="activeTutorial.icon"
                          :icon-svg="activeTutorial.icon_svg"
                          size="md"
                        />
                      </span>
                      <div class="min-w-0">
                        <p class="text-sm font-medium text-gray-900 dark:text-white">
                          {{ activeTutorial.icon_svg ? t('admin.documentation.customIconActive') : t('admin.documentation.builtInIconActive') }}
                        </p>
                        <p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-dark-400">
                          {{ t('admin.documentation.customIconPriorityHint') }}
                        </p>
                      </div>
                    </div>

                    <div class="mt-3 border-t border-gray-200 pt-3 dark:border-dark-700">
                      <label :for="`tutorial-icon-${activeTutorial.id}`" class="mb-1 block text-xs font-medium text-gray-600 dark:text-dark-300">
                        {{ t('admin.documentation.fallbackIcon') }}
                      </label>
                      <div class="relative">
                        <DocumentationCategoryIcon
                          :icon="activeTutorial.icon"
                          class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-dark-300"
                        />
                        <select :id="`tutorial-icon-${activeTutorial.id}`" v-model="activeTutorial.icon" class="input pl-9">
                          <option v-for="option in iconOptions" :key="option.value" :value="option.value">
                            {{ option.label }}
                          </option>
                        </select>
                      </div>
                    </div>

                    <div class="mt-3 border-t border-gray-200 pt-3 dark:border-dark-700">
                      <p class="mb-2 text-xs font-medium text-gray-600 dark:text-dark-300">
                        {{ t('admin.documentation.customIcon') }}
                      </p>
                      <ImageUpload
                        :model-value="activeTutorial.icon_svg ?? ''"
                        mode="svg"
                        size="sm"
                        :show-preview="false"
                        :max-size="DOCUMENTATION_MAX_ICON_SVG_BYTES"
                        :upload-label="t('admin.documentation.uploadIconSvg')"
                        :remove-label="t('admin.documentation.removeIconSvg')"
                        :hint="t('admin.documentation.iconSvgHint')"
                        :max-size-error-message="t('admin.documentation.iconSvgTooLarge')"
                        :read-error-message="t('admin.documentation.iconSvgReadFailed')"
                        @update:model-value="updateActiveTutorialIconSvg"
                      />
                    </div>
                  </div>
                </div>
                <div>
                  <label :for="`tutorial-label-${activeTutorial.id}`" class="input-label">
                    {{ t('admin.documentation.tabLabel') }}
                  </label>
                  <input :id="`tutorial-label-${activeTutorial.id}`" v-model="activeTutorial.tab_label" class="input" maxlength="80" :placeholder="t('admin.documentation.tabLabelPlaceholder')" />
                </div>
              </div>
              <div>
                <label :for="`tutorial-id-${activeTutorial.id}`" class="input-label">
                  {{ t('admin.documentation.categoryId') }}
                </label>
                <input
                  :id="`tutorial-id-${activeTutorial.id}`"
                  :value="activeTutorial.id"
                  class="input font-mono text-sm"
                  maxlength="64"
                  :placeholder="t('admin.documentation.categoryIdPlaceholder')"
                  @input="updateActiveTutorialId"
                />
                <p class="input-hint">{{ t('admin.documentation.categoryIdHint') }}</p>
              </div>
              <div>
                <label :for="`tutorial-description-${activeTutorial.id}`" class="input-label">
                  {{ t('admin.documentation.categoryDescription') }}
                </label>
                <textarea :id="`tutorial-description-${activeTutorial.id}`" v-model="activeTutorial.description" class="input min-h-24" rows="3" maxlength="500" :placeholder="t('admin.documentation.categoryDescriptionPlaceholder')"></textarea>
              </div>
            </fieldset>

            <div class="border-t border-gray-200 pt-6 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h3 class="text-sm font-semibold text-gray-950 dark:text-white">
                    {{ t('admin.documentation.steps') }}
                  </h3>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                    {{ t('admin.documentation.stepsHint') }}
                  </p>
                </div>
                <button type="button" class="btn btn-secondary" @click="addStep">
                  <Icon name="plus" size="sm" aria-hidden="true" />
                  <span class="ml-1.5">{{ t('admin.documentation.addStep') }}</span>
                </button>
              </div>

              <ol v-if="activeTutorial.steps.length" class="mt-4 divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-700 dark:border-dark-700">
                <li v-for="(step, stepIndex) in activeTutorial.steps" :key="`${activeTutorial.id}-${stepIndex}`" class="py-6 first:pt-5 last:pb-5">
                  <article>
                    <div class="mb-4 flex items-center gap-3">
                      <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-950 text-sm font-semibold text-white dark:bg-white dark:text-gray-950" aria-hidden="true">
                        {{ stepIndex + 1 }}
                      </span>
                      <h4 class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-950 dark:text-white">
                        {{ step.title || t('admin.documentation.untitledStep') }}
                      </h4>
                      <div class="flex items-center gap-1">
                        <button type="button" class="editor-icon-button" :disabled="stepIndex === 0" :title="t('admin.documentation.moveUp')" @click="moveStep(stepIndex, -1)">
                          <Icon name="arrowUp" size="sm" aria-hidden="true" />
                        </button>
                        <button type="button" class="editor-icon-button" :disabled="stepIndex === activeTutorial.steps.length - 1" :title="t('admin.documentation.moveDown')" @click="moveStep(stepIndex, 1)">
                          <Icon name="arrowDown" size="sm" aria-hidden="true" />
                        </button>
                        <button type="button" class="editor-icon-button text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20" :title="t('admin.documentation.deleteStep')" @click="requestDeleteStep(stepIndex)">
                          <Icon name="trash" size="sm" aria-hidden="true" />
                        </button>
                      </div>
                    </div>

                    <div class="space-y-4 pl-0 sm:pl-11">
                      <div>
                        <label :for="`step-title-${activeTutorial.id}-${stepIndex}`" class="input-label">
                          {{ t('admin.documentation.stepTitle') }}
                        </label>
                        <input :id="`step-title-${activeTutorial.id}-${stepIndex}`" v-model="step.title" class="input" maxlength="120" :placeholder="t('admin.documentation.stepTitlePlaceholder')" />
                      </div>
                      <div>
                        <label :for="`step-description-${activeTutorial.id}-${stepIndex}`" class="input-label">
                          {{ t('admin.documentation.stepDescription') }}
                        </label>
                        <textarea :id="`step-description-${activeTutorial.id}-${stepIndex}`" v-model="step.description" class="input min-h-24" rows="3" maxlength="1200" :placeholder="t('admin.documentation.stepDescriptionPlaceholder')"></textarea>
                      </div>

                      <div class="flex flex-wrap gap-2" :aria-label="t('admin.documentation.optionalFeatures')">
                        <button v-if="!step.note" type="button" class="feature-button" @click="addNote(step)">
                          <Icon name="infoCircle" size="xs" aria-hidden="true" />
                          {{ t('admin.documentation.addNote') }}
                        </button>
                        <button v-if="!step.code" type="button" class="feature-button" @click="addCode(step)">
                          <Icon name="terminal" size="xs" aria-hidden="true" />
                          {{ t('admin.documentation.addCode') }}
                        </button>
                        <button v-if="!step.image" type="button" class="feature-button" @click="addImage(step)">
                          <Icon name="upload" size="xs" aria-hidden="true" />
                          {{ t('admin.documentation.addImage') }}
                        </button>
                        <button v-if="!step.link" type="button" class="feature-button" @click="addLink(step)">
                          <Icon name="link" size="xs" aria-hidden="true" />
                          {{ t('admin.documentation.addLink') }}
                        </button>
                      </div>

                      <fieldset v-if="step.note" class="optional-section">
                        <div class="optional-section-header">
                          <legend class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.documentation.note') }}</legend>
                          <button type="button" class="remove-feature-button" @click="removeFeature(step, 'note')">{{ t('admin.documentation.removeSection') }}</button>
                        </div>
                        <div class="grid gap-3 sm:grid-cols-2">
                          <div>
                            <label :for="`note-tone-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.noteTone') }}</label>
                            <select :id="`note-tone-${activeTutorial.id}-${stepIndex}`" v-model="step.note.tone" class="input">
                              <option value="info">{{ t('admin.documentation.noteInfo') }}</option>
                              <option value="warning">{{ t('admin.documentation.noteWarning') }}</option>
                            </select>
                          </div>
                          <div>
                            <label :for="`note-placement-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.notePlacement') }}</label>
                            <select :id="`note-placement-${activeTutorial.id}-${stepIndex}`" v-model="step.note_placement" class="input">
                              <option value="before-image">{{ t('admin.documentation.beforeImage') }}</option>
                              <option value="after-image">{{ t('admin.documentation.afterImage') }}</option>
                            </select>
                          </div>
                        </div>
                        <div class="mt-3">
                          <label :for="`note-text-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.noteText') }}</label>
                          <textarea :id="`note-text-${activeTutorial.id}-${stepIndex}`" v-model="step.note.text" class="input min-h-20" rows="2"></textarea>
                        </div>
                      </fieldset>

                      <fieldset v-if="step.code" class="optional-section">
                        <div class="optional-section-header">
                          <legend class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.documentation.code') }}</legend>
                          <button type="button" class="remove-feature-button" @click="removeFeature(step, 'code')">{{ t('admin.documentation.removeSection') }}</button>
                        </div>
                        <div>
                          <label :for="`code-label-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.codeLabel') }}</label>
                          <input :id="`code-label-${activeTutorial.id}-${stepIndex}`" v-model="step.code.label" class="input" maxlength="80" />
                        </div>
                        <div class="mt-3">
                          <label :for="`code-value-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.codeValue') }}</label>
                          <textarea :id="`code-value-${activeTutorial.id}-${stepIndex}`" v-model="step.code.value" class="input min-h-36 font-mono text-sm" rows="6" spellcheck="false"></textarea>
                        </div>
                      </fieldset>

                      <fieldset v-if="step.image" class="optional-section">
                        <div class="optional-section-header">
                          <legend class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.documentation.image') }}</legend>
                          <button type="button" class="remove-feature-button" @click="removeFeature(step, 'image')">{{ t('admin.documentation.removeSection') }}</button>
                        </div>
                        <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                          <div class="min-w-0 flex-1">
                            <label :for="`image-src-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.imageUrl') }}</label>
                            <input :id="`image-src-${activeTutorial.id}-${stepIndex}`" v-model="step.image.src" class="input" type="url" placeholder="https://…" />
                          </div>
                          <label class="btn btn-secondary cursor-pointer" :class="{ 'pointer-events-none opacity-60': uploadingStepKey === stepKey(stepIndex) }">
                            <Icon name="upload" size="sm" aria-hidden="true" />
                            <span class="ml-1.5">{{ uploadingStepKey === stepKey(stepIndex) ? t('admin.documentation.uploading') : t('admin.documentation.uploadImage') }}</span>
                            <input class="sr-only" type="file" accept="image/png,image/jpeg,image/webp,image/gif" :disabled="uploadingStepKey === stepKey(stepIndex)" @change="uploadStepImage($event, step, stepIndex)" />
                          </label>
                        </div>
                        <div class="mt-3 grid gap-3 sm:grid-cols-2">
                          <div>
                            <label :for="`image-alt-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.imageAlt') }}</label>
                            <input :id="`image-alt-${activeTutorial.id}-${stepIndex}`" v-model="step.image.alt" class="input" maxlength="180" />
                          </div>
                          <div>
                            <label :for="`image-caption-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.imageCaption') }}</label>
                            <input :id="`image-caption-${activeTutorial.id}-${stepIndex}`" v-model="step.image.caption" class="input" maxlength="240" />
                          </div>
                        </div>
                        <img v-if="safeImageUrl(step.image.src)" :src="safeImageUrl(step.image.src)" :alt="step.image.alt" class="mt-3 max-h-48 max-w-full rounded border border-gray-200 object-contain dark:border-dark-700" />
                      </fieldset>

                      <fieldset v-if="step.link" class="optional-section">
                        <div class="optional-section-header">
                          <legend class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.documentation.link') }}</legend>
                          <button type="button" class="remove-feature-button" @click="removeFeature(step, 'link')">{{ t('admin.documentation.removeSection') }}</button>
                        </div>
                        <div class="grid gap-3 sm:grid-cols-2">
                          <div>
                            <label :for="`link-label-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.linkLabel') }}</label>
                            <input :id="`link-label-${activeTutorial.id}-${stepIndex}`" v-model="step.link.label" class="input" maxlength="100" />
                          </div>
                          <div>
                            <label :for="`link-href-${activeTutorial.id}-${stepIndex}`" class="input-label">{{ t('admin.documentation.linkUrl') }}</label>
                            <input :id="`link-href-${activeTutorial.id}-${stepIndex}`" v-model="step.link.href" class="input" type="url" placeholder="https://…" />
                          </div>
                        </div>
                      </fieldset>
                    </div>
                  </article>
                </li>
              </ol>

              <div v-else class="mt-4 rounded-xl border border-dashed border-gray-300 px-6 py-10 text-center dark:border-dark-600">
                <Icon name="document" size="lg" class="mx-auto text-gray-400" aria-hidden="true" />
                <p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.documentation.noSteps') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.documentation.noStepsHint') }}</p>
                <button type="button" class="btn btn-secondary mt-4" @click="addStep">
                  {{ t('admin.documentation.addFirstStep') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="rounded-2xl border border-gray-200 bg-white px-6 py-20 text-center dark:border-dark-700 dark:bg-dark-800">
          <Icon name="book" size="xl" class="mx-auto text-gray-400" aria-hidden="true" />
          <h2 class="mt-3 text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.documentation.noCategories') }}</h2>
          <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('admin.documentation.noCategoriesHint') }}</p>
          <button type="button" class="btn btn-primary mt-5" @click="addCategory">{{ t('admin.documentation.addCategory') }}</button>
        </div>
      </section>
    </div>

    <BaseDialog :show="showPreview" :title="t('admin.documentation.previewTitle')" width="extra-wide" @close="showPreview = false">
      <DocumentationPreview :content="content" />
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showPreview = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showHistory" :title="t('admin.documentation.historyTitle')" width="normal" @close="showHistory = false">
      <div v-if="revisionsLoading" class="space-y-2" aria-live="polite">
        <div v-for="index in 4" :key="index" class="h-16 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700"></div>
      </div>
      <div v-else-if="revisionsError" class="rounded-xl border border-red-200 bg-red-50 px-4 py-4 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-200">
        <p>{{ revisionsError }}</p>
        <button type="button" class="mt-2 font-medium underline underline-offset-2" @click="loadRevisions">{{ t('admin.documentation.retry') }}</button>
      </div>
      <ol v-else-if="revisions.length" class="divide-y divide-gray-200 dark:divide-dark-700">
        <li v-for="revision in revisions" :key="revision.id" class="flex items-center justify-between gap-4 py-3.5 first:pt-0 last:pb-0">
          <div class="min-w-0">
            <p class="text-sm font-medium text-gray-950 dark:text-white">
              {{ t('admin.documentation.versionLabel', { version: revision.version ?? revision.id }) }}
            </p>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
              {{ formatTimestamp(revision.published_at || revision.created_at || revision.restored_at) }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary flex-shrink-0" @click="requestRestoreRevision(revision)">
            {{ t('admin.documentation.restoreToDraft') }}
          </button>
        </li>
      </ol>
      <div v-else class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('admin.documentation.noRevisions') }}
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="confirmation !== null"
      :title="confirmationTitle"
      :message="confirmationMessage"
      :confirm-text="confirmationConfirmText"
      :danger="confirmation?.kind === 'delete-category' || confirmation?.kind === 'delete-step'"
      @confirm="confirmAction"
      @cancel="confirmation = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  DocumentationContent,
  DocumentationIconKey,
  DocumentationRevision,
  DocumentationSnapshot,
  DocumentationStep,
  DocumentationTutorial,
} from '@/api/admin/documentation'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Icon from '@/components/icons/Icon.vue'
import DocumentationCategoryIcon from '@/components/admin/documentation/DocumentationCategoryIcon.vue'
import DocumentationPreview from '@/components/admin/documentation/DocumentationPreview.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { sanitizeDocumentationIconSvg } from '@/utils/documentationSvg'
import { formatDateTime } from '@/utils/format'
import { sanitizeUrl } from '@/utils/url'

type OptionalFeature = 'note' | 'code' | 'image' | 'link'
const DOCUMENTATION_MAX_ASSET_BYTES = 5 * 1024 * 1024
const DOCUMENTATION_MAX_ICON_SVG_BYTES = 32 * 1024
const DOCUMENTATION_ICON_KEYS = new Set<DocumentationIconKey>([
  'key', 'client', 'api', 'wallet', 'image', 'help', 'book', 'code', 'terminal',
])
type Confirmation =
  | { kind: 'publish' }
  | { kind: 'restore'; revision: DocumentationRevision }
  | { kind: 'delete-category' }
  | { kind: 'delete-step'; stepIndex: number }

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const publishing = ref(false)
const content = ref<DocumentationContent>({ schema_version: 1, tutorials: [] })
const publishedSnapshot = ref<DocumentationSnapshot | null>(null)
const draftUpdatedAt = ref<string | null>(null)
const savedFingerprint = ref('')
const activeTutorialId = ref('')
const validationErrors = ref<string[]>([])
const validationAlert = ref<HTMLElement | null>(null)
const uploadingStepKey = ref('')

const showPreview = ref(false)
const showHistory = ref(false)
const revisions = ref<DocumentationRevision[]>([])
const revisionsLoading = ref(false)
const revisionsError = ref('')
const revisionsLoaded = ref(false)
const confirmation = ref<Confirmation | null>(null)

const fingerprint = computed(() => JSON.stringify(content.value))
const isDirty = computed(() => fingerprint.value !== savedFingerprint.value)
const activeTutorialIndex = computed(() => content.value.tutorials.findIndex((item) => item.id === activeTutorialId.value))
const activeTutorial = computed(() => content.value.tutorials[activeTutorialIndex.value] ?? null)

const confirmationTitle = computed(() => {
  switch (confirmation.value?.kind) {
    case 'publish': return t('admin.documentation.publishTitle')
    case 'restore': return t('admin.documentation.restoreTitle')
    case 'delete-category': return t('admin.documentation.deleteCategoryTitle')
    case 'delete-step': return t('admin.documentation.deleteStepTitle')
    default: return ''
  }
})

const confirmationMessage = computed(() => {
  switch (confirmation.value?.kind) {
    case 'publish': return t('admin.documentation.publishConfirm')
    case 'restore': return t('admin.documentation.restoreConfirm', { version: confirmation.value.revision.version ?? confirmation.value.revision.id })
    case 'delete-category': return t('admin.documentation.deleteCategoryConfirm', { name: activeTutorial.value?.tab_label || t('admin.documentation.untitledCategory') })
    case 'delete-step': return t('admin.documentation.deleteStepConfirm')
    default: return ''
  }
})

const confirmationConfirmText = computed(() => {
  switch (confirmation.value?.kind) {
    case 'publish': return t('admin.documentation.publish')
    case 'restore': return t('admin.documentation.restoreToDraft')
    case 'delete-category':
    case 'delete-step': return t('common.delete')
    default: return t('common.confirm')
  }
})

function cloneContent(value: DocumentationContent): DocumentationContent {
  return JSON.parse(JSON.stringify(value)) as DocumentationContent
}

function normalizeDocumentationIconSvg(value: string | undefined): string {
  return sanitizeDocumentationIconSvg(value)
}

function createId(prefix: string): string {
  const random = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID().slice(0, 8)
    : Math.random().toString(36).slice(2, 10)
  return `${prefix}-${random}`
}

function normalizeContent(value: DocumentationContent): DocumentationContent {
  const preserved = cloneContent(value)
  return {
    ...preserved,
    schema_version: 1,
    tutorials: (value.tutorials ?? []).map((tutorial, tutorialIndex) => {
      const iconSvg = normalizeDocumentationIconSvg(tutorial.icon_svg)
      return {
        id: tutorial.id || `tutorial-${tutorialIndex + 1}`,
        tab_label: tutorial.tab_label ?? '',
        icon: DOCUMENTATION_ICON_KEYS.has(tutorial.icon) ? tutorial.icon : 'help',
        ...(iconSvg ? { icon_svg: iconSvg } : {}),
        description: tutorial.description ?? '',
        steps: (tutorial.steps ?? []).map((step) => ({
          title: step.title ?? '',
          description: step.description ?? '',
          ...(step.note ? { note: { tone: step.note.tone || 'info', text: step.note.text ?? '' } } : {}),
          ...(step.note ? { note_placement: step.note_placement || 'before-image' } : {}),
          ...(step.code ? { code: { label: step.code.label ?? '', value: step.code.value ?? '' } } : {}),
          ...(step.image ? { image: { src: step.image.src ?? '', alt: step.image.alt ?? '', ...(step.image.caption !== undefined ? { caption: step.image.caption } : {}) } } : {}),
          ...(step.link ? { link: { label: step.link.label ?? '', href: step.link.href ?? '' } } : {}),
        })),
      }
    }),
  }
}

async function loadDocumentation(): Promise<void> {
  if (isDirty.value && !loading.value) {
    appStore.showWarning(t('admin.documentation.reloadDirtyWarning'))
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const state = await adminAPI.documentation.get()
    const draftContent = normalizeContent(state.draft.content)
    content.value = cloneContent(draftContent)
    savedFingerprint.value = JSON.stringify(draftContent)
    publishedSnapshot.value = state.published
    draftUpdatedAt.value = state.draft.updated_at ?? null
    activeTutorialId.value = draftContent.tutorials[0]?.id ?? ''
    validationErrors.value = []
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, t('admin.documentation.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function persistDraft(showSuccess = true): Promise<boolean> {
  saving.value = true
  try {
    const snapshot = await adminAPI.documentation.saveDraft(cloneContent(content.value))
    savedFingerprint.value = fingerprint.value
    draftUpdatedAt.value = snapshot.updated_at ?? new Date().toISOString()
    validationErrors.value = []
    if (showSuccess) appStore.showSuccess(t('admin.documentation.saveSuccess'))
    return true
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.documentation.saveFailed')))
    return false
  } finally {
    saving.value = false
  }
}

async function saveDraft(): Promise<void> {
  await persistDraft(true)
}

function requestPublish(): void {
  validationErrors.value = validateForPublish()
  if (validationErrors.value.length) {
    nextTick(() => {
      const alert = validationAlert.value
      alert?.focus()
      if (typeof alert?.scrollIntoView === 'function') {
        alert.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }
    })
    return
  }
  confirmation.value = { kind: 'publish' }
}

async function publish(): Promise<void> {
  publishing.value = true
  try {
    if (isDirty.value && !(await persistDraft(false))) return
    const state = await adminAPI.documentation.publish()
    publishedSnapshot.value = state.published ?? state.draft
    appStore.showSuccess(t('admin.documentation.publishSuccess'))
    revisionsLoaded.value = false
    await loadFreshAfterMutation()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.documentation.publishFailed')))
  } finally {
    publishing.value = false
  }
}

async function loadFreshAfterMutation(): Promise<void> {
  try {
    const state = await adminAPI.documentation.get()
    const draftContent = normalizeContent(state.draft.content)
    content.value = cloneContent(draftContent)
    savedFingerprint.value = JSON.stringify(draftContent)
    publishedSnapshot.value = state.published
    draftUpdatedAt.value = state.draft.updated_at ?? null
    if (!draftContent.tutorials.some((tutorial) => tutorial.id === activeTutorialId.value)) {
      activeTutorialId.value = draftContent.tutorials[0]?.id ?? ''
    }
  } catch {
    // The mutation already succeeded; keep the local draft and let manual refresh retry metadata.
  }
}

function addCategory(): void {
  const tutorial: DocumentationTutorial = {
    id: createId('tutorial'),
    tab_label: t('admin.documentation.newCategory'),
    icon: 'help',
    description: '',
    steps: [],
  }
  content.value.tutorials.push(tutorial)
  activeTutorialId.value = tutorial.id
}

function updateActiveTutorialId(event: Event): void {
  const nextId = (event.target as HTMLInputElement).value.trim()
  const tutorial = content.value.tutorials[activeTutorialIndex.value]
  if (!tutorial) return
  tutorial.id = nextId
  activeTutorialId.value = nextId
}

function updateActiveTutorialIconSvg(value: string): void {
  const tutorial = activeTutorial.value
  if (!tutorial) return
  if (!value.trim()) {
    delete tutorial.icon_svg
    return
  }

  const sanitized = normalizeDocumentationIconSvg(value)
  if (!sanitized) {
    appStore.showError(t('admin.documentation.iconSvgInvalid'))
    return
  }
  tutorial.icon_svg = sanitized
}

function moveCategory(offset: -1 | 1): void {
  const from = activeTutorialIndex.value
  const to = from + offset
  if (from < 0 || to < 0 || to >= content.value.tutorials.length) return
  const [tutorial] = content.value.tutorials.splice(from, 1)
  content.value.tutorials.splice(to, 0, tutorial)
}

function requestDeleteCategory(): void {
  confirmation.value = { kind: 'delete-category' }
}

function deleteCategory(): void {
  const index = activeTutorialIndex.value
  if (index < 0) return
  content.value.tutorials.splice(index, 1)
  activeTutorialId.value = content.value.tutorials[Math.min(index, content.value.tutorials.length - 1)]?.id ?? ''
}

function addStep(): void {
  activeTutorial.value?.steps.push({ title: '', description: '' })
}

function moveStep(stepIndex: number, offset: -1 | 1): void {
  const steps = activeTutorial.value?.steps
  if (!steps) return
  const targetIndex = stepIndex + offset
  if (targetIndex < 0 || targetIndex >= steps.length) return
  const [step] = steps.splice(stepIndex, 1)
  steps.splice(targetIndex, 0, step)
}

function requestDeleteStep(stepIndex: number): void {
  confirmation.value = { kind: 'delete-step', stepIndex }
}

function deleteStep(stepIndex: number): void {
  activeTutorial.value?.steps.splice(stepIndex, 1)
}

function addNote(step: DocumentationStep): void {
  step.note = { tone: 'info', text: '' }
  step.note_placement = 'before-image'
}

function addCode(step: DocumentationStep): void {
  step.code = { label: '', value: '' }
}

function addImage(step: DocumentationStep): void {
  step.image = { src: '', alt: '', caption: '' }
}

function addLink(step: DocumentationStep): void {
  step.link = { label: '', href: '' }
}

function removeFeature(step: DocumentationStep, feature: OptionalFeature): void {
  delete step[feature]
  if (feature === 'note') delete step.note_placement
}

function stepKey(stepIndex: number): string {
  return `${activeTutorial.value?.id ?? 'tutorial'}:${stepIndex}`
}

async function uploadStepImage(event: Event, step: DocumentationStep, stepIndex: number): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !step.image) return
  if (file.size > DOCUMENTATION_MAX_ASSET_BYTES) {
    appStore.showError(t('admin.documentation.uploadTooLarge'))
    input.value = ''
    return
  }
  const key = stepKey(stepIndex)
  uploadingStepKey.value = key
  try {
    const result = await adminAPI.documentation.uploadAsset(file)
    if (!result.url) throw new Error(t('admin.documentation.uploadMissingUrl'))
    step.image.src = result.url
    if (!step.image.alt) step.image.alt = file.name.replace(/\.[^.]+$/, '')
    appStore.showSuccess(t('admin.documentation.uploadSuccess'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.documentation.uploadFailed')))
  } finally {
    uploadingStepKey.value = ''
    input.value = ''
  }
}

function validateForPublish(): string[] {
  const errors: string[] = []
  if (!content.value.tutorials.length) errors.push(t('admin.documentation.validation.categoriesRequired'))
  const ids = new Set<string>()
  content.value.tutorials.forEach((tutorial, tutorialIndex) => {
    const categoryName = tutorial.tab_label.trim() || t('admin.documentation.categoryNumber', { number: tutorialIndex + 1 })
    if (!tutorial.id.trim()) errors.push(t('admin.documentation.validation.categoryIdRequired', { category: categoryName }))
    if (tutorial.id && !/^[a-z][a-z0-9-]{0,63}$/.test(tutorial.id)) {
      errors.push(t('admin.documentation.validation.categoryIdInvalid', { id: tutorial.id }))
    }
    if (tutorial.id && ids.has(tutorial.id)) errors.push(t('admin.documentation.validation.categoryIdDuplicate', { id: tutorial.id }))
    ids.add(tutorial.id)
    if (!tutorial.tab_label.trim()) errors.push(t('admin.documentation.validation.categoryLabelRequired', { number: tutorialIndex + 1 }))
    if (!tutorial.description.trim()) errors.push(t('admin.documentation.validation.categoryDescriptionRequired', { category: categoryName }))
    if (!tutorial.steps.length) errors.push(t('admin.documentation.validation.stepsRequired', { category: categoryName }))
    tutorial.steps.forEach((step, stepIndex) => {
      const stepNumber = stepIndex + 1
      if (!step.title.trim()) errors.push(t('admin.documentation.validation.stepTitleRequired', { category: categoryName, step: stepNumber }))
      if (!step.description.trim()) errors.push(t('admin.documentation.validation.stepDescriptionRequired', { category: categoryName, step: stepNumber }))
      if (step.note && !step.note.text.trim()) errors.push(t('admin.documentation.validation.noteTextRequired', { category: categoryName, step: stepNumber }))
      if (step.code && (!step.code.label.trim() || !step.code.value.trim())) {
        errors.push(t('admin.documentation.validation.codeRequired', { category: categoryName, step: stepNumber }))
      }
      if (step.image && !sanitizeUrl(step.image.src, { allowRelative: true })) {
        errors.push(t('admin.documentation.validation.imageUrlRequired', { category: categoryName, step: stepNumber }))
      }
      if (step.image && !step.image.alt.trim()) errors.push(t('admin.documentation.validation.imageAltRequired', { category: categoryName, step: stepNumber }))
      if (step.link && (!step.link.label.trim() || !sanitizeUrl(step.link.href, { allowRelative: true }))) {
        errors.push(t('admin.documentation.validation.linkInvalid', { category: categoryName, step: stepNumber }))
      }
    })
  })
  return errors
}

async function openHistory(): Promise<void> {
  showHistory.value = true
  if (!revisionsLoaded.value) await loadRevisions()
}

async function loadRevisions(): Promise<void> {
  revisionsLoading.value = true
  revisionsError.value = ''
  try {
    const response = await adminAPI.documentation.revisions()
    revisions.value = response.items
    revisionsLoaded.value = true
  } catch (error) {
    revisionsError.value = extractApiErrorMessage(error, t('admin.documentation.historyLoadFailed'))
  } finally {
    revisionsLoading.value = false
  }
}

function requestRestoreRevision(revision: DocumentationRevision): void {
  confirmation.value = { kind: 'restore', revision }
}

async function restoreRevision(revision: DocumentationRevision): Promise<void> {
  try {
    await adminAPI.documentation.restoreRevision(revision.id)
    appStore.showSuccess(t('admin.documentation.restoreSuccess'))
    showHistory.value = false
    revisionsLoaded.value = false
    await loadFreshAfterMutation()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.documentation.restoreFailed')))
  }
}

async function confirmAction(): Promise<void> {
  const action = confirmation.value
  confirmation.value = null
  if (!action) return
  if (action.kind === 'publish') await publish()
  if (action.kind === 'restore') await restoreRevision(action.revision)
  if (action.kind === 'delete-category') deleteCategory()
  if (action.kind === 'delete-step') deleteStep(action.stepIndex)
}

function safeImageUrl(value: string): string {
  return sanitizeUrl(value, { allowRelative: true })
}

const iconOptions = computed(() => [
  { value: 'key', label: t('admin.documentation.icons.key') },
  { value: 'client', label: t('admin.documentation.icons.client') },
  { value: 'api', label: t('admin.documentation.icons.api') },
  { value: 'wallet', label: t('admin.documentation.icons.wallet') },
  { value: 'image', label: t('admin.documentation.icons.image') },
  { value: 'help', label: t('admin.documentation.icons.help') },
  { value: 'book', label: t('admin.documentation.icons.book') },
  { value: 'code', label: t('admin.documentation.icons.code') },
  { value: 'terminal', label: t('admin.documentation.icons.terminal') },
])

function formatTimestamp(value: string | null | undefined): string {
  return value ? formatDateTime(value) : '—'
}

function handleBeforeUnload(event: BeforeUnloadEvent): void {
  if (!isDirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  loadDocumentation()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})

onBeforeRouteLeave(() => {
  if (!isDirty.value) return true
  return window.confirm(t('admin.documentation.leaveDirtyConfirm'))
})
</script>

<style scoped>
.documentation-category-sidebar {
  position: static;
}

@media (min-width: 1024px) {
  .documentation-category-sidebar {
    position: sticky;
    top: calc(var(--app-shell-top-offset) + 20px);
  }
}

.editor-icon-button {
  @apply inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 disabled:cursor-not-allowed disabled:opacity-35 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-white;
}

.feature-button {
  @apply inline-flex min-h-8 items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-2.5 text-xs font-medium text-gray-700 transition-colors hover:border-primary-300 hover:bg-primary-50 hover:text-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-800 dark:hover:bg-primary-900/20 dark:hover:text-primary-200;
}

.optional-section {
  @apply rounded-xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/30;
}

.optional-section-header {
  @apply mb-3 flex items-center justify-between gap-3;
}

.remove-feature-button {
  @apply rounded-md px-1.5 py-1 text-xs font-medium text-red-600 hover:bg-red-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500/40 dark:text-red-400 dark:hover:bg-red-900/20;
}

@media (prefers-reduced-motion: reduce) {
  .editor-icon-button,
  .feature-button {
    transition-duration: 0.01ms;
  }
}
</style>
