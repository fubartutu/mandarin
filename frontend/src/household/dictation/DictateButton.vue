<template>
	<!-- A control of its own next to the field; nothing at all where the browser can't dictate. -->
	<p
		v-if="supported"
		class="control"
	>
		<XButton
			class="dictate-button"
			:class="{'is-listening': listening}"
			variant="secondary"
			:icon="listening ? 'stop' : 'microphone'"
			:aria-label="listening ? $t('household.dictation.stop') : $t('household.dictation.start')"
			:aria-pressed="listening"
			:title="listening ? $t('household.dictation.stop') : $t('household.dictation.start')"
			@click="toggle"
		/>
	</p>
</template>

<script setup lang="ts">
import {onBeforeUnmount, ref} from 'vue'

import {error} from '@/message'
import {i18n} from '@/i18n'
import {createRecognition, errorKeyFor, isDictationSupported, joinDictated, transcriptOf, type Recognition} from './dictation'

// The text field's value: `<DictateButton v-model="newTaskTitle" />`. What is
// said is added after what's already there, updating as the words come in.
const model = defineModel<string>({default: ''})

const supported = isDictationSupported()
const listening = ref(false)
let recognition: Recognition | null = null

function start() {
	// The phone's language (e.g. en-AU), so Australian spellings and accents work.
	recognition = createRecognition(navigator.language)
	if (recognition === null) {
		return
	}
	const before = model.value
	recognition.onresult = (event) => {
		model.value = joinDictated(before, transcriptOf(event.results))
	}
	recognition.onerror = (event) => {
		const key = errorKeyFor(event.error)
		if (key !== null) {
			error(new Error(i18n.global.t(`household.dictation.error.${key}`)))
		}
	}
	recognition.onend = () => {
		listening.value = false
		recognition = null
	}
	// Must run straight from the tap: browsers ask for the microphone then.
	recognition.start()
	listening.value = true
}

function toggle() {
	if (recognition !== null) {
		// stop() keeps what was heard; onend resets the button.
		recognition.stop()
		return
	}
	start()
}

onBeforeUnmount(() => recognition?.abort())
</script>

<style lang="scss" scoped>
.dictate-button {
	block-size: 100% !important;

	:deep(.icon) {
		margin: 0 !important;
	}

	&.is-listening {
		color: var(--danger);
		animation: dictate-pulse 1.2s ease-in-out infinite;
	}
}

@keyframes dictate-pulse {
	50% {
		opacity: .55;
	}
}

@media (prefers-reduced-motion: reduce) {
	.dictate-button.is-listening {
		animation: none;
	}
}
</style>
