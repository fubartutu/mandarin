<template>
	<FormField
		:label="$t('household.startPage.label')"
		layout="two-col"
	>
		<FormSelect
			v-model="page"
			:options="options"
		/>
	</FormField>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import FormField from '@/components/input/FormField.vue'
import FormSelect from '@/components/input/FormSelect.vue'
import {START_PAGES, startPageFrom} from '../startPage'

// defineModel() is Vue's two-way binding: the settings form passes
// `v-model="settings.frontend_settings.household_start_page"`.
const model = defineModel<string>({default: ''})

const {t} = useI18n({useScope: 'global'})

// Unset shows as Home; picking Home saves 'home'.
const page = computed({
	get: () => startPageFrom(model.value),
	set: (value: string) => {
		model.value = startPageFrom(value)
	},
})

const options = computed(() => START_PAGES.map(value => ({value, label: t(`household.startPage.${value}`)})))
</script>
