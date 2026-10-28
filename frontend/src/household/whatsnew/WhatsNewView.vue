<template>
	<div class="whats-new">
		<Card :title="$t('household.whatsNew.title')">
			<p class="version has-text-grey">
				{{ $t('household.whatsNew.version', {version}) }}
			</p>

			<section
				v-for="entry in changelog"
				:key="entry.date"
				class="entry"
			>
				<h3 class="subtitle">
					{{ dayLabel(entry.date) }}
				</h3>
				<ul>
					<li
						v-for="(change, i) in entry.changes"
						:key="i"
					>
						<span
							class="kind"
							:class="`is-${change.kind}`"
						>{{ $t(`household.whatsNew.kind.${change.kind}`) }}</span>
						{{ change.text }}
					</li>
				</ul>
			</section>
		</Card>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'

import Card from '@/components/misc/Card.vue'
import {useConfigStore} from '@/stores/config'
import {VERSION as frontendVersion} from '@/version.json'

import {changelog, entryDate} from './changelog'
import {useWhatsNew} from './seen'

// The server's version, e.g. v2.7.1-112-82f8702b1: the Vikunja release we're
// based on, our commits since then and the commit itself. Falls back to the
// frontend's own (the same in the Docker image) before /info has loaded.
const configStore = useConfigStore()
const version = computed(() => configStore.version || frontendVersion)

// "Tuesday 27 October 2026" in the app's language; Vikunja's long format adds a time.
const {locale} = useI18n()
function dayLabel(date: string) {
	return entryDate(date).toLocaleDateString(locale.value, {weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'})
}

// Opening this page clears the "what's new" notice on Home.
const {markSeen} = useWhatsNew()
onMounted(markSeen)
</script>

<style lang="scss" scoped>
.whats-new {
	max-inline-size: 45rem;
	margin-inline: auto;
}

.version {
	margin-block-end: 1.5rem;
	font-size: .875rem;
}

.entry + .entry {
	margin-block-start: 1.5rem;
}

.subtitle {
	margin-block-end: .5rem;
	font-size: 1rem;
	font-weight: 600;
}

ul {
	margin: 0;
	list-style: none;
}

// Tag in its own column, so wrapped lines start under the text, not the tag.
li {
	display: flex;
	gap: .6rem;
	align-items: baseline;
	margin-block-end: .4rem;
	line-height: 1.5;
}

.kind {
	flex-shrink: 0;
	min-inline-size: 4.5rem;
	padding: 0 .4rem;
	border-radius: 999px;
	font-size: .75rem;
	text-align: center;

	&.is-new {
		background: var(--primary-light);
		color: var(--primary-dark);
	}

	&.is-improved {
		background: var(--grey-200);
		color: var(--grey-800);
	}

	&.is-fixed {
		background: var(--success-light, var(--grey-200));
		color: var(--success-dark, var(--grey-800));
	}
}
</style>
