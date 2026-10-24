<template>
	<div class="household-home">
		<h1
			v-if="salutation"
			class="title has-text-centered"
		>
			{{ salutation }}
		</h1>

		<Message
			v-if="deletionScheduledAt !== null"
			variant="danger"
			class="mbe-4"
		>
			{{
				$t('user.deletion.scheduled', {
					date: formatDisplayDate(deletionScheduledAt),
					dateSince: formatDateSince(deletionScheduledAt),
				})
			}}
			<RouterLink :to="{name: 'user.settings.deletion'}">
				{{ $t('user.deletion.scheduledCancel') }}
			</RouterLink>
		</Message>

		<AddTask @tasksAdded="showTasksKey++" />

		<div class="home-cards">
			<HomeMenuCard />

			<Card
				:title="$t('household.shopping.title')"
				class="home-card"
			>
				<p
					v-if="shoppingLists.length === 0"
					class="has-text-grey is-italic"
				>
					{{ shopping.isPending.value ? $t('household.home.loading') : $t('household.shopping.noLists') }}
				</p>
				<ul
					v-else
					class="list-rows"
				>
					<li
						v-for="list in shoppingLists"
						:key="list.id"
					>
						<RouterLink
							:to="{name: 'shopping', params: {listId: list.id}}"
							class="list-row"
						>
							<span class="list-title">
								{{ list.title }}
								<span class="count">{{ $t('household.home.toBuy', {count: list.open.length}, list.open.length) }}</span>
							</span>
							<span
								v-if="list.open.length > 0"
								class="preview"
							>
								{{ preview(list.open) }}
							</span>
						</RouterLink>
					</li>
				</ul>
				<template #footer>
					<RouterLink
						:to="{name: 'shopping'}"
						class="card-footer-item"
					>
						{{ $t('household.home.openShopping') }}
					</RouterLink>
				</template>
			</Card>

			<Card
				:title="recentRecipes.viewed ? $t('household.home.recentRecipes') : $t('household.home.newRecipes')"
				class="home-card"
			>
				<p
					v-if="recentRecipes.list.length === 0"
					class="has-text-grey is-italic"
				>
					{{ recipes.isPending.value ? $t('household.home.loading') : $t('household.recipes.empty') }}
				</p>
				<ul
					v-else
					class="recipe-rows"
				>
					<li
						v-for="recipe in recentRecipes.list"
						:key="recipe.id"
					>
						<RouterLink
							:to="{name: 'recipe', params: {recipeId: recipe.id}}"
							class="recipe-row"
						>
							<RecipeImage
								class="thumb"
								:recipe-id="recipe.id"
								:image-id="recipe.image_ids[0]"
								alt=""
							/>
							<span>{{ recipe.title }}</span>
						</RouterLink>
					</li>
				</ul>
				<template #footer>
					<RouterLink
						:to="{name: 'recipes'}"
						class="card-footer-item"
					>
						{{ $t('household.home.allRecipes') }}
					</RouterLink>
				</template>
			</Card>
		</div>

		<!-- Vikunja's "Current tasks": overdue and due soon, across every project. -->
		<ShowTasks
			v-if="projectList.hasProjects"
			:key="showTasksKey"
			:label-ids="labelIds"
			class="home-tasks"
			@clearLabelFilter="clearLabelFilter"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useQuery} from '@tanstack/vue-query'

import AddTask from '@/components/tasks/AddTask.vue'
import Card from '@/components/misc/Card.vue'
import Message from '@/components/misc/Message.vue'
import ShowTasks from '@/views/tasks/ShowTasks.vue'
import {useDaytimeSalutation} from '@/composables/useDaytimeSalutation'
import {useProjects} from '@/composables/useProjects'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {formatDateSince, formatDisplayDate} from '@/helpers/time/formatDate'
import {useAuthStore} from '@/stores/auth'

import HomeMenuCard from './menu/components/HomeMenuCard.vue'
import RecipeImage from './recipes/components/RecipeImage.vue'
import {recipesQuery} from './recipes/queries'
import {useRecentRecipes} from './recipes/recent'
import type {ShoppingItemResponse} from './shopping/queries'
import {useShoppingData} from './shopping/useShopping'
import {useShoppingSync} from './shopping/sync/useShoppingSync'

// Replaces Vikunja's Overview (views/Home.vue) on "/": the same greeting,
// quick add and current tasks, with this week's menu, the shopping lists and recipes in place
// of the "last viewed projects" cards. Registered in setup.ts.

const PREVIEW_ITEMS = 4

const salutation = useDaytimeSalutation()
const authStore = useAuthStore()
const projectList = useProjects()
const route = useRoute()
const router = useRouter()

const deletionScheduledAt = computed(() => parseDateOrNull(authStore.info?.deletion_scheduled_at))

// ---- Shopping: read from the phone's copy, so this part works offline ----

const shopping = useShoppingData()
const sync = useShoppingSync()

const shoppingLists = computed(() => {
	const items = [...sync.data.value.items.values()]
	return shopping.lists.value.map(list => ({
		...list,
		open: items.filter(i => i.list_id === list.id && !i.done).sort((a, b) => a.position - b.position),
	}))
})

function preview(items: ShoppingItemResponse[]): string {
	const names = items.slice(0, PREVIEW_ITEMS).map(i => i.name).join(', ')
	const more = items.length - PREVIEW_ITEMS
	return more > 0 ? `${names} +${more}` : names
}

// ---- Recipes: the ones opened lately, or the newest if none yet ----

const recipes = useQuery(recipesQuery())
const recentIds = useRecentRecipes()

const recentRecipes = computed(() => {
	const all = recipes.data.value ?? []
	const byId = new Map(all.map(r => [r.id, r]))
	const viewed = recentIds.value.map(id => byId.get(id)).filter(r => r !== undefined)
	if (viewed.length > 0) return {viewed: true, list: viewed.slice(0, PREVIEW_ITEMS)}
	return {viewed: false, list: [...all].sort((a, b) => b.id - a.id).slice(0, PREVIEW_ITEMS)}
})

// ---- Tasks ----

// Bumped after a quick add so ShowTasks loads again.
const showTasksKey = ref(0)

// ?labels=1,2 filters the current tasks, as on Vikunja's Overview.
const labelIds = computed(() => {
	const labels = route.query.labels
	if (!labels) return undefined
	return Array.isArray(labels) ? labels.filter((id): id is string => id !== null) : [labels]
})

function clearLabelFilter() {
	const query = {...route.query}
	delete query.labels
	router.push({name: 'home', query})
}
</script>

<style lang="scss" scoped>
.household-home {
	max-inline-size: $desktop;
	margin-inline: auto;
}

.home-cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
	gap: 1rem;
	margin-block-start: 1.5rem;
}

.home-card {
	display: flex;
	flex-direction: column;
	margin: 0;

	:deep(.card-content) {
		flex: 1;
	}
}

.list-rows,
.recipe-rows {
	margin: 0;
	padding: 0;
	list-style: none;
}

.list-row {
	display: flex;
	flex-direction: column;
	padding: .4rem 0;
	color: var(--text);
}

li + li .list-row {
	border-block-start: 1px solid var(--grey-200);
}

.list-title {
	font-weight: 600;
}

.count {
	margin-inline-start: .4rem;
	color: var(--grey-500);
	font-size: .85rem;
	font-weight: normal;
}

.preview {
	overflow: hidden;
	color: var(--grey-600);
	font-size: .9rem;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.recipe-row {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .3rem 0;
	color: var(--text);
	font-weight: 600;
}

.thumb {
	flex: none;
	inline-size: 3rem;
	block-size: 3rem;
	border-radius: $radius;
	overflow: hidden;
	font-size: 1.25rem;
}

.home-tasks {
	margin-block-start: 2rem;
}
</style>
