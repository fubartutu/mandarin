<template>
	<Card :title="$t('household.push.title')">
		<p class="intro">
			{{ $t('household.push.intro') }}
		</p>

		<!-- This device -->
		<section class="this-device">
			<h3 class="subtitle">
				{{ $t('household.push.thisDevice') }}
			</h3>
			<p
				v-if="state !== 'on' && state !== 'off'"
				class="help is-warning"
			>
				{{ $t(`household.push.state.${state}`) }}
			</p>
			<p
				v-else-if="state === 'on'"
				class="help is-success"
			>
				{{ $t('household.push.state.on') }}
			</p>
			<div class="buttons">
				<XButton
					v-if="state === 'off'"
					icon="bell"
					:loading="busy"
					@click="change(turnOn)"
				>
					{{ $t('household.push.turnOn') }}
				</XButton>
				<XButton
					v-if="state === 'on'"
					variant="secondary"
					:loading="busy"
					@click="change(turnOff)"
				>
					{{ $t('household.push.turnOff') }}
				</XButton>
				<XButton
					v-if="(devices.data.value ?? []).length > 0"
					variant="secondary"
					:loading="test.isPending.value"
					@click="test.mutate()"
				>
					{{ $t('household.push.test') }}
				</XButton>
			</div>
		</section>

		<!-- Every device of mine -->
		<section v-if="(devices.data.value ?? []).length > 0">
			<h3 class="subtitle">
				{{ $t('household.push.devices') }}
			</h3>
			<ul class="device-list">
				<li
					v-for="device in devices.data.value"
					:key="device.id"
				>
					<span>{{ device.name || $t('household.push.unnamed') }}</span>
					<span class="device-when">{{ device.last_sent
						? $t('household.push.lastSent', {when: formatDateSince(device.last_sent)})
						: $t('household.push.added', {when: formatDateSince(device.created)}) }}</span>
					<BaseButton
						class="device-remove"
						:aria-label="$t('household.push.remove', {name: device.name})"
						@click="removeDevice.mutate(device.id ?? 0)"
					>
						<Icon icon="times" />
					</BaseButton>
				</li>
			</ul>
		</section>

		<!-- What pushes -->
		<section v-if="prefs">
			<h3 class="subtitle">
				{{ $t('household.push.what') }}
			</h3>
			<FormCheckbox
				v-model="prefs.reminders"
				:label="$t('household.push.reminders')"
			/>
			<FormCheckbox
				v-model="prefs.overdue"
				:label="$t('household.push.overdue')"
			/>
			<p class="help">
				{{ $t('household.push.overdueHint') }}
				<RouterLink :to="{name: 'user.settings.general'}">
					{{ $t('user.settings.general.title') }}
				</RouterLink>
			</p>

			<p class="label shopping-label">
				{{ $t('household.push.shopping') }}
			</p>
			<FormCheckbox
				v-for="list in lists"
				:key="list.id"
				:model-value="prefs.shopping_list_ids.includes(list.id)"
				:label="list.title"
				@update:modelValue="toggleList(list.id, $event)"
			/>
			<p class="help">
				{{ $t('household.push.shoppingHint') }}
			</p>

			<FormCheckbox
				v-model="prefs.quiet_hours"
				:label="$t('household.push.quietHours')"
			/>
			<p class="help">
				{{ $t('household.push.quietHoursHint') }}
			</p>

			<XButton
				class="save"
				:loading="savePrefs.isPending.value"
				:disabled="!dirty"
				@click="savePrefs.mutate(prefs)"
			>
				{{ $t('misc.save') }}
			</XButton>
		</section>
	</Card>
</template>

<script setup lang="ts">
import {computed, onMounted, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useMutation, useQuery} from '@tanstack/vue-query'

import BaseButton from '@/components/base/BaseButton.vue'
import FormCheckbox from '@/components/input/FormCheckbox.vue'
import Card from '@/components/misc/Card.vue'
import {pushDevicesDelete, pushDevicesList, pushSettingsGet, pushSettingsUpdate, pushTest} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {useTitle} from '@/composables/useTitle'
import {formatDateSince} from '@/helpers/time/formatDate'
import {error, success} from '@/message'

import {useShoppingData} from '../shopping/useShopping'
import {deviceState, turnOff, turnOn, type DeviceState} from './device'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('household.push.title'))

const keys = {
	devices: ['push', 'devices'] as const,
	settings: ['push', 'settings'] as const,
}

// ---- This device ----

const state = ref<DeviceState>('off')
const busy = ref(false)
onMounted(async () => {
	state.value = await deviceState().catch(() => 'unsupported' as const)
})

async function change(action: () => Promise<DeviceState>) {
	busy.value = true
	try {
		state.value = await action()
		await devices.refetch()
	} catch (e) {
		error(e)
	} finally {
		busy.value = false
	}
}

// ---- Devices ----

const devices = useQuery({
	queryKey: keys.devices,
	queryFn: async () => (await pushDevicesList()).data.devices ?? [],
})

const removeDevice = useMutation(contextMutationOptions({
	mutationFn: async (id: number) => {
		await pushDevicesDelete({path: {device: id}})
	},
	onSettled: (_input, client) => client.invalidateQueries({queryKey: keys.devices}),
}))

const test = useMutation(contextMutationOptions({
	mutationFn: async () => (await pushTest()).data.sent ?? 0,
	onSuccess: sent => success({message: t('household.push.testSent', {count: sent}, sent)}),
	onSettled: (_input, client) => client.invalidateQueries({queryKey: keys.devices}),
}))

// ---- Preferences ----

interface Prefs {
	reminders: boolean
	overdue: boolean
	quiet_hours: boolean
	shopping_list_ids: number[]
}

const saved = useQuery({
	queryKey: keys.settings,
	queryFn: async (): Promise<Prefs> => {
		const p = (await pushSettingsGet()).data
		return {
			reminders: p.reminders ?? true,
			overdue: p.overdue ?? true,
			quiet_hours: p.quiet_hours ?? true,
			shopping_list_ids: p.shopping_list_ids ?? [],
		}
	},
})
// An editable copy; saved explicitly.
const prefs = ref<Prefs | null>(null)
watch(() => saved.data.value, value => {
	if (value) prefs.value = {...value, shopping_list_ids: [...value.shopping_list_ids]}
}, {immediate: true})
const dirty = computed(() => JSON.stringify(prefs.value) !== JSON.stringify(saved.data.value))

const {lists} = useShoppingData()
function toggleList(id: number, on: boolean) {
	if (!prefs.value) return
	const others = prefs.value.shopping_list_ids.filter(x => x !== id)
	prefs.value.shopping_list_ids = on ? [...others, id] : others
}

const savePrefs = useMutation(contextMutationOptions({
	mutationFn: async (p: Prefs) => (await pushSettingsUpdate({body: p})).data,
	successMessage: () => t('household.push.saved'),
	onSettled: (_input, client) => client.invalidateQueries({queryKey: keys.settings}),
}))
</script>

<style lang="scss" scoped>
.intro {
	margin-block-end: 1rem;
}

section {
	margin-block-end: 1.5rem;
}

.subtitle {
	margin-block-end: .5rem;
	font-size: 1.05rem;
	font-weight: 600;
}

.buttons {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	margin-block-start: .5rem;
}

.device-list {
	margin: 0;
	padding: 0;
	list-style: none;

	li {
		display: flex;
		align-items: center;
		gap: .75rem;
		padding-block: .4rem;
		border-block-end: 1px solid var(--grey-200);
	}
}

.device-when {
	margin-inline-start: auto;
	color: var(--grey-500);
	font-size: .85rem;
}

.device-remove {
	padding: .25rem .5rem;
	color: var(--grey-500);
}

.shopping-label {
	margin-block: 1rem .25rem;
}

.help {
	margin-block-end: .75rem;
}

.save {
	margin-block-start: .5rem;
}
</style>
