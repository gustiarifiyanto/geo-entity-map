import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ApiError } from './api/client'
import App from './App.tsx'
import { ToastProvider } from './components/toast/ToastProvider'
import { I18nProvider } from './i18n/I18nProvider'
import { isSessionExpired, markLoggedOut } from './features/auth/hooks'
import 'leaflet/dist/leaflet.css'
import './index.css'

// Any request that finds the session missing or expired sends the user back
// to the login screen.
const onError = (error: unknown) => {
  if (isSessionExpired(error)) markLoggedOut(queryClient)
}

const queryClient: QueryClient = new QueryClient({
  queryCache: new QueryCache({ onError }),
  mutationCache: new MutationCache({ onError }),
  defaultOptions: {
    queries: {
      // Retrying a 4xx will not change the outcome.
      retry: (failureCount, error) =>
        !(error instanceof ApiError && error.status >= 400 && error.status < 500) &&
        failureCount < 2,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <App />
        </ToastProvider>
      </QueryClientProvider>
    </I18nProvider>
  </StrictMode>,
)
