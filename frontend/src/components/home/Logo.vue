<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '@vueuse/core'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useColorScheme } from '@/composables/useColorScheme'

// FORK: Mandarin logo (no pride variant, so both point at it)
import LogoFull from '@/household/components/MandarinLogo.vue'
import LogoFullPride from '@/household/components/MandarinLogo.vue'
import {APP_NAME} from '@/household/branding'
import {MILLISECONDS_A_HOUR} from '@/constants/date'

const now = useNow({
	interval: MILLISECONDS_A_HOUR,
})

const authStore = useAuthStore()
const configStore = useConfigStore()
const { isDark } = useColorScheme()

const Logo = computed(() => configStore.allow_icon_changes
	&& authStore.settings.frontend_settings.allow_icon_changes
	&& now.value.getMonth() === 5
	? LogoFullPride
	: LogoFull)

const CustomLogo = computed(() => {
	const lightLogo = window.CUSTOM_LOGO_URL
	const darkLogo = window.CUSTOM_LOGO_URL_DARK

	if (!lightLogo && !darkLogo) return ''
	if (!darkLogo) return lightLogo
	if (!lightLogo) return darkLogo

	return isDark.value ? darkLogo : lightLogo
})
</script>

<template>
	<div>
		<!-- FORK: alt text uses APP_NAME -->
		<Logo
			v-if="!CustomLogo"
			:alt="APP_NAME"
			class="logo"
		/>
		<img
			v-else
			:src="CustomLogo"
			:alt="APP_NAME"
			class="logo"
		>
	</div>
</template>

<style lang="scss" scoped>
.logo {
	color: var(--logo-text-color);
	max-inline-size: 168px;
	max-block-size: 48px;
}
</style>
