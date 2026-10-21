<template>
	<div class="recipe-editor">
		<h1 class="title">
			{{ recipeId ? $t('household.recipes.editTitle') : $t('household.recipes.newTitle') }}
		</h1>

		<!-- Import: only for a new recipe. The result fills the form below for checking. -->
		<section
			v-if="!recipeId"
			class="import"
		>
			<div
				class="import-source"
				role="tablist"
			>
				<button
					v-for="option in (['web', 'paperless'] as const)"
					:key="option"
					type="button"
					role="tab"
					class="source-tab"
					:class="{'is-active': source === option}"
					:aria-selected="source === option"
					@click="source = option"
				>
					{{ $t(`household.recipes.source.${option}`) }}
				</button>
			</div>

			<form
				v-if="source === 'web'"
				@submit.prevent="runImport"
			>
				<label
					class="label"
					for="recipe-import-url"
				>{{ $t('household.recipes.importLabel') }}</label>
				<div class="import-row">
					<input
						id="recipe-import-url"
						v-model="importUrl"
						class="input"
						type="url"
						inputmode="url"
						:placeholder="$t('household.recipes.importPlaceholder')"
						required
					>
					<XButton
						type="submit"
						:loading="importing.isPending.value"
						:disabled="!online"
					>
						{{ $t('household.recipes.import') }}
					</XButton>
				</div>
			</form>
			<PaperlessPicker
				v-else-if="online"
				:busy="importingScan.isPending.value"
				@pick="docId => runScanImport(docId, false)"
			/>

			<p
				v-if="!online"
				class="help"
			>
				{{ $t('household.recipes.needsSignal') }}
			</p>
			<p
				v-else-if="importingScan.isPending.value || (importing.isPending.value && source === 'web')"
				class="help"
			>
				{{ $t('household.recipes.reading') }}
			</p>
			<div
				v-else-if="importInfo"
				class="import-status"
				:class="importInfo.complete ? 'is-success' : 'is-warning'"
			>
				<span>{{ $t(`household.recipes.importedBy.${importInfo.complete ? importInfo.method : 'incomplete'}`) }}</span>
				<XButton
					v-if="importInfo.scanId && importInfo.method !== 'ai' && importInfo.aiAvailable"
					variant="secondary"
					:loading="importingScan.isPending.value"
					@click="runScanImport(importInfo.scanId, true)"
				>
					{{ $t('household.recipes.tryAI') }}
				</XButton>
			</div>
		</section>

		<p
			v-if="recipeId && existing.isPending.value"
			class="has-text-grey"
		>
			{{ $t('household.recipes.loading') }}
		</p>

		<form
			v-else
			class="fields"
			@submit.prevent="save"
		>
			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.title') }}</span>
				<input
					v-model="form.title"
					class="input"
					type="text"
					maxlength="250"
					required
				>
			</label>

			<label class="field">
				<span class="label">{{ $t('household.recipes.fields.servings') }}</span>
				<input
					v-model.number="form.servings"
					class="input"
					type="number"
					min="0"
					step="any"
					inputmode="decimal"
				>
			</label>
			<label class="field">
				<span class="label">{{ $t('household.recipes.fields.yields') }}</span>
				<input
					v-model="form.yields"
					class="input"
					type="text"
					maxlength="100"
					:placeholder="$t('household.recipes.fields.yieldsPlaceholder')"
				>
			</label>
			<label class="field">
				<span class="label">{{ $t('household.recipes.fields.prep') }}</span>
				<input
					v-model.number="form.prep_minutes"
					class="input"
					type="number"
					min="0"
					inputmode="numeric"
				>
			</label>
			<label class="field">
				<span class="label">{{ $t('household.recipes.fields.cook') }}</span>
				<input
					v-model.number="form.cook_minutes"
					class="input"
					type="number"
					min="0"
					inputmode="numeric"
				>
			</label>
			<label class="field">
				<span class="label">{{ $t('household.recipes.fields.total') }}</span>
				<input
					v-model.number="form.total_minutes"
					class="input"
					type="number"
					min="0"
					inputmode="numeric"
				>
			</label>

			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.ingredients') }}</span>
				<textarea
					v-model="form.ingredientsText"
					class="textarea"
					rows="10"
					:placeholder="$t('household.recipes.fields.ingredientsPlaceholder')"
				/>
				<span class="help">{{ $t('household.recipes.fields.ingredientsHelp') }}</span>
			</label>

			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.steps') }}</span>
				<textarea
					v-model="form.stepsText"
					class="textarea"
					rows="10"
				/>
				<span class="help">{{ $t('household.recipes.fields.stepsHelp') }}</span>
			</label>

			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.tags') }}</span>
				<input
					v-model="form.tagsText"
					class="input"
					type="text"
					:placeholder="$t('household.recipes.fields.tagsPlaceholder')"
				>
			</label>
			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.description') }}</span>
				<textarea
					v-model="form.description"
					class="textarea"
					rows="2"
					maxlength="5000"
				/>
			</label>
			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.notes') }}</span>
				<textarea
					v-model="form.notes"
					class="textarea"
					rows="3"
					maxlength="10000"
					:placeholder="$t('household.recipes.fields.notesPlaceholder')"
				/>
			</label>
			<label class="field is-wide">
				<span class="label">{{ $t('household.recipes.fields.source') }}</span>
				<input
					v-model="form.source_url"
					class="input"
					type="url"
					maxlength="2000"
				>
			</label>

			<!-- Pictures the import found: picked ones are downloaded by the server on save. -->
			<fieldset
				v-if="!recipeId && foundImages.length > 0"
				class="field is-wide"
			>
				<legend class="label">
					{{ $t('household.recipes.foundPictures') }}
				</legend>
				<div class="picture-row">
					<label
						v-for="url in foundImages"
						:key="url"
						class="found-picture"
					>
						<PaperlessThumb
							v-if="paperlessDocumentId(url)"
							:doc-id="paperlessDocumentId(url)"
						/>
						<img
							v-else
							:src="url"
							alt=""
							referrerpolicy="no-referrer"
						>
						<input
							v-model="pickedImages"
							type="checkbox"
							:value="url"
						>
					</label>
				</div>
			</fieldset>

			<!-- Saved pictures: only when editing (a new recipe needs an id first). -->
			<fieldset
				v-if="recipeId && existing.data.value"
				class="field is-wide"
			>
				<legend class="label">
					{{ $t('household.recipes.pictures') }}
				</legend>
				<div class="picture-row">
					<div
						v-for="(id, i) in existing.data.value.image_ids"
						:key="id"
						class="saved-picture"
					>
						<RecipeImage
							:recipe-id="recipeId"
							:image-id="id"
							alt=""
						/>
						<div class="picture-actions">
							<BaseButton
								v-if="i > 0"
								:disabled="!online"
								@click="makeMain.mutate({recipeId, imageId: id})"
							>
								{{ $t('household.recipes.makeMain') }}
							</BaseButton>
							<BaseButton
								:disabled="!online"
								@click="deleteImage.mutate({recipeId, imageId: id})"
							>
								{{ $t('household.recipes.removePicture') }}
							</BaseButton>
						</div>
					</div>
					<label
						v-if="existing.data.value.image_ids.length < 4"
						class="button add-picture"
						:disabled="!online || undefined"
					>
						<Icon icon="camera" />
						<span>{{ $t('household.recipes.addPicture') }}</span>
						<input
							class="is-sr-only"
							type="file"
							accept="image/*"
							:disabled="!online"
							@change="uploadPicture"
						>
					</label>
				</div>
			</fieldset>

			<div class="actions is-wide">
				<XButton
					v-if="recipeId"
					variant="secondary"
					class="is-danger"
					icon="trash-alt"
					:disabled="!online"
					@click="remove"
				>
					{{ $t('household.recipes.delete') }}
				</XButton>
				<XButton
					variant="tertiary"
					@click="router.back()"
				>
					{{ $t('household.recipes.cancel') }}
				</XButton>
				<XButton
					type="submit"
					:loading="saving"
					:disabled="!online"
				>
					{{ $t('household.recipes.save') }}
				</XButton>
			</div>
		</form>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, reactive, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useRoute, useRouter} from 'vue-router'
import {useQuery} from '@tanstack/vue-query'
import {useOnline} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import {useTitle} from '@/composables/useTitle'
import {error} from '@/message'

import PaperlessPicker from './components/PaperlessPicker.vue'
import PaperlessThumb from './components/PaperlessThumb.vue'
import RecipeImage from './components/RecipeImage.vue'
import {
	paperlessDocumentId,
	recipeQuery,
	useAddImageMutation,
	useDeleteImageMutation,
	useDeleteRecipeMutation,
	useImportPaperlessMutation,
	useImportRecipeMutation,
	useMakeImageMainMutation,
	useSaveRecipeMutation,
	type RecipeDraft,
	type RecipeResponse,
} from './queries'
import {ingredientsToText, textToIngredients} from './scaling'

const props = defineProps<{
	recipeId: number // 0 for a new recipe
}>()

const {t} = useI18n({useScope: 'global'})
const router = useRouter()
const route = useRoute()
const online = useOnline()
useTitle(() => props.recipeId ? t('household.recipes.editTitle') : t('household.recipes.newTitle'))

// The form holds ingredients, steps and tags as text; see scaling.ts for
// the ingredient format (heading lines end in ":").
const form = reactive({
	title: '',
	source_url: '',
	description: '',
	servings: 0,
	yields: '',
	prep_minutes: 0,
	cook_minutes: 0,
	total_minutes: 0,
	ingredientsText: '',
	stepsText: '',
	tagsText: '',
	notes: '',
})

function fill(r: RecipeResponse) {
	Object.assign(form, {
		title: r.title,
		source_url: r.source_url,
		description: r.description,
		servings: r.servings,
		yields: r.yields,
		prep_minutes: r.prep_minutes,
		cook_minutes: r.cook_minutes,
		total_minutes: r.total_minutes,
		ingredientsText: ingredientsToText(r.ingredients),
		stepsText: r.steps.join('\n'),
		tagsText: r.tags.join(', '),
		notes: r.notes,
	})
}

// Editing: fill the form once the recipe has loaded (not on every refetch,
// which would wipe what's been typed).
const existing = useQuery(computed(() => ({...recipeQuery(props.recipeId), enabled: props.recipeId > 0})))
let filled = false
watch(() => existing.data.value, r => {
	if (r && !filled) {
		fill(r)
		filled = true
	}
}, {immediate: true})

// ---- Import ----

const importUrl = ref(typeof route.query.url === 'string' ? route.query.url : '')
const source = ref<'web' | 'paperless'>('web')
const importing = useImportRecipeMutation()
const importingScan = useImportPaperlessMutation()
const foundImages = ref<string[]>([])
const pickedImages = ref<string[]>([])

// How the last import was read, for the line under the import box.
const importInfo = ref<{method: string, complete: boolean, aiAvailable: boolean, scanId: number} | null>(null)

function useImported(r: RecipeResponse, scanId = 0) {
	fill(r)
	foundImages.value = r.image_urls
	pickedImages.value = r.image_urls.slice(0, 1)
	importInfo.value = {method: r.import_method, complete: r.import_complete, aiAvailable: r.ai_available, scanId}
}

async function runImport() {
	try {
		useImported(await importing.mutateAsync(importUrl.value.trim()))
	} catch {
		importInfo.value = null // the error is already toasted
	}
}

async function runScanImport(docId: number, aiFirst: boolean) {
	try {
		useImported(await importingScan.mutateAsync({docId, aiFirst}), docId)
	} catch {
		// toasted; a failed "try the AI" keeps the rules' attempt in the form
	}
}

// Opened as /recipes/new?url=…: import straight away.
onMounted(() => {
	if (!props.recipeId && importUrl.value) runImport()
})

// ---- Save ----

const saveRecipe = useSaveRecipeMutation()
const addImage = useAddImageMutation()
const saving = ref(false)

function draft(): RecipeDraft {
	return {
		title: form.title,
		source_url: form.source_url,
		description: form.description,
		servings: Number(form.servings) || 0,
		yields: form.yields,
		prep_minutes: Number(form.prep_minutes) || 0,
		cook_minutes: Number(form.cook_minutes) || 0,
		total_minutes: Number(form.total_minutes) || 0,
		ingredients: textToIngredients(form.ingredientsText),
		steps: form.stepsText.split('\n').map(s => s.trim()).filter(Boolean),
		tags: form.tagsText.split(',').map(s => s.trim()).filter(Boolean),
		notes: form.notes,
	}
}

async function save() {
	saving.value = true
	try {
		const saved = await saveRecipe.mutateAsync({id: props.recipeId, draft: draft()})
		// One at a time: the server stores each picture in its own write.
		for (const url of pickedImages.value) {
			await addImage.mutateAsync({recipeId: saved.id, url}).catch(() => undefined)
		}
		router.replace({name: 'recipe', params: {recipeId: saved.id}})
	} catch {
		// toasted by the mutation
	} finally {
		saving.value = false
	}
}

// ---- Pictures and delete (editing) ----

const makeMain = useMakeImageMainMutation()
const deleteImage = useDeleteImageMutation()

function uploadPicture(event: Event) {
	const input = event.target as HTMLInputElement
	const file = input.files?.[0]
	if (file) addImage.mutate({recipeId: props.recipeId, file})
	input.value = '' // so picking the same file again still fires
}

const deleteRecipe = useDeleteRecipeMutation()
async function remove() {
	if (!window.confirm(t('household.recipes.deleteConfirm', {title: form.title}))) return
	try {
		await deleteRecipe.mutateAsync(props.recipeId)
		router.replace({name: 'recipes'})
	} catch (e) {
		error(e)
	}
}
</script>

<style lang="scss" scoped>
.recipe-editor {
	max-inline-size: 48rem;
	margin-inline: auto;
}

.import {
	margin-block-end: 1.5rem;
	padding: 1rem;
	border-radius: $radius;
	background: var(--grey-100);
}

.import-row {
	display: flex;
	gap: .5rem;
}

.import-source {
	display: flex;
	gap: .25rem;
	margin-block-end: .75rem;
}

.source-tab {
	padding: .35rem .8rem;
	border: 0;
	border-radius: 999px;
	background: transparent;
	color: var(--grey-600);
	cursor: pointer;

	&.is-active {
		background: var(--primary);
		color: var(--white);
	}
}

.import-status {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem 1rem;
	margin-block-start: .5rem;
	font-size: .9rem;

	&.is-success {
		color: var(--success-dark);
	}

	&.is-warning {
		color: var(--warning-dark);
	}
}

.fields {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
	gap: .75rem 1rem;

	.is-wide {
		grid-column: 1 / -1;
	}
}

.field {
	margin: 0;
	padding: 0;
	border: 0;

	.label {
		display: block;
		margin-block-end: .25rem;
		font-size: .85rem;
	}

	.input, .textarea {
		font-size: 1rem;
	}
}

.picture-row {
	display: flex;
	flex-wrap: wrap;
	gap: .75rem;
	align-items: flex-start;
}

.found-picture {
	position: relative;
	inline-size: 8rem;
	cursor: pointer;

	img, :deep(.paperless-thumb) {
		display: block;
		inline-size: 100%;
		aspect-ratio: 4 / 3;
		object-fit: cover;
		border-radius: $radius;
	}

	input {
		position: absolute;
		inset-block-start: .4rem;
		inset-inline-start: .4rem;
		inline-size: 1.2rem;
		block-size: 1.2rem;
	}
}

.saved-picture {
	inline-size: 8rem;

	:deep(.recipe-image) {
		aspect-ratio: 4 / 3;
		block-size: auto;
		border-radius: $radius;
	}
}

.picture-actions {
	display: flex;
	flex-wrap: wrap;
	gap: .25rem .75rem;
	margin-block-start: .25rem;
	color: var(--primary);
	font-size: .8rem;

	button:disabled {
		opacity: .4;
	}
}

.actions {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: .5rem;

	.is-danger {
		margin-inline-end: auto;
	}
}
</style>
