import {CancelledError, QueryCache, QueryClient} from '@tanstack/vue-query'

import {isRequestContextAbort} from '@/client/requestContext'
import {error} from '@/message'
import {isQuietOfflineError} from '@/household/offline' // FORK: Mandarin offline shopping

export const queryClient = new QueryClient({
	queryCache: new QueryCache({
		onError: (cause, query) => {
			if (query.meta?.handlesError) return
			if (cause instanceof CancelledError || isRequestContextAbort(cause)) return
			if (isQuietOfflineError(cause)) return // FORK: the shopping list shows its own offline badge
			error(cause)
		},
	}),
	defaultOptions: {
		queries: {
			staleTime: 60_000,
			retry: 1,
			refetchOnWindowFocus: false,
		},
		mutations: {
			retry: false,
		},
	},
})
