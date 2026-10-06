import { zodResolver } from '@hookform/resolvers/zod'
import { useState, type ReactNode } from 'react'
import { useForm, type UseFormSetError } from 'react-hook-form'
import { ApiError } from '../../api/client'
import { inputClass } from '../../components/formStyles'
import {
  loginSchema,
  PASSWORD_MIN,
  registerSchema,
  type CredentialsFormValues,
} from '../../schemas/auth'
import { useLogin, useRegister } from './hooks'

type Mode = 'login' | 'register'

const MODES: Record<Mode, { tab: string; title: string; submit: string; submitting: string }> = {
  login: { tab: 'Log in', title: 'Log in to your account', submit: 'Log in', submitting: 'Logging in…' },
  register: {
    tab: 'Register',
    title: 'Create an account',
    submit: 'Create account',
    submitting: 'Creating account…',
  },
}

/** Shown while logged out. New accounts can view the map; only admins can change it. */
export function AuthScreen() {
  const [mode, setMode] = useState<Mode>('login')

  return (
    <main className="flex min-h-full items-center justify-center bg-gray-100 p-4">
      <div className="w-full max-w-sm animate-fade-in-up rounded-xl bg-white shadow-xl ring-1 ring-black/5">
        <header className="border-b border-gray-100 p-6 pb-4">
          <h1 className="text-lg font-semibold text-gray-900">Geo Entity Map</h1>
          <p className="mt-1 text-sm text-gray-500">{MODES[mode].title}</p>
          <div role="tablist" aria-label="Account" className="mt-4 grid grid-cols-2 rounded-lg bg-gray-100 p-1 text-sm">
            {(Object.keys(MODES) as Mode[]).map((m) => (
              <button
                key={m}
                type="button"
                role="tab"
                aria-selected={mode === m}
                onClick={() => setMode(m)}
                className={`rounded-md px-3 py-1.5 font-medium ${
                  mode === m ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {MODES[m].tab}
              </button>
            ))}
          </div>
        </header>
        {/* Remount on switch so values and errors from the other form do not carry over. */}
        <CredentialsForm key={mode} mode={mode} />
      </div>
    </main>
  )
}

function CredentialsForm({ mode }: { mode: Mode }) {
  const login = useLogin()
  const register = useRegister()
  const [showPassword, setShowPassword] = useState(false)
  const {
    register: field,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<CredentialsFormValues>({
    resolver: zodResolver(mode === 'login' ? loginSchema : registerSchema),
    defaultValues: { email: '', password: '' },
    // Validate on submit only: the email field is autofocused, so validating on
    // blur would show "is required" as soon as the user clicks anywhere else.
    // After the first submit, errors update as the user types.
    mode: 'onSubmit',
    reValidateMode: 'onChange',
  })

  const submit = handleSubmit(async (values) => {
    try {
      // On success the app switches to the map, so nothing else happens here.
      await (mode === 'login' ? login : register).mutateAsync(values)
    } catch (error) {
      applyServerError(error, setError)
    }
  })

  const text = MODES[mode]
  return (
    <form onSubmit={submit} noValidate aria-label={text.title} className="space-y-4 p-6">
      {errors.root?.server && (
        <div role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 ring-1 ring-red-200">
          {errors.root.server.message}
        </div>
      )}

      <FormField id="email" label="Email" error={errors.email?.message}>
        <input
          id="email"
          type="email"
          autoFocus
          autoComplete={mode === 'login' ? 'username' : 'email'}
          aria-invalid={errors.email ? true : undefined}
          className={inputClass}
          {...field('email')}
        />
      </FormField>

      <FormField
        id="password"
        label="Password"
        error={errors.password?.message}
        hint={mode === 'register' ? `At least ${PASSWORD_MIN} characters.` : undefined}
      >
        <div className="relative">
          <input
            id="password"
            type={showPassword ? 'text' : 'password'}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
            aria-invalid={errors.password ? true : undefined}
            className={`${inputClass} pr-16`}
            {...field('password')}
          />
          <button
            type="button"
            onClick={() => setShowPassword((show) => !show)}
            aria-controls="password"
            aria-pressed={showPassword}
            className="absolute inset-y-0 right-0 px-3 text-xs font-medium text-gray-500 hover:text-gray-900"
          >
            {showPassword ? 'Hide' : 'Show'}
          </button>
        </div>
      </FormField>

      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full rounded-lg bg-gray-900 px-3 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-60"
      >
        {isSubmitting ? text.submitting : text.submit}
      </button>

      {mode === 'register' && (
        <p className="text-center text-xs text-gray-500">
          New accounts can view the map. Ask an admin if you need to manage entities.
        </p>
      )}
    </form>
  )
}

/**
 * 422 field errors go under their inputs; a wrong email/password and anything
 * else become a form-level error.
 */
function applyServerError(error: unknown, setError: UseFormSetError<CredentialsFormValues>) {
  if (error instanceof ApiError && error.isValidation) {
    let first = true
    for (const name of ['email', 'password'] as const) {
      const message = error.fields[name]
      if (message) {
        setError(name, { type: 'server', message }, { shouldFocus: first })
        first = false
      }
    }
    if (!first) return
  }

  const message =
    error instanceof ApiError && error.code === 'invalid_credentials'
      ? 'Incorrect email or password.'
      : error instanceof Error
        ? error.message
        : 'Something went wrong. Please try again.'
  setError('root.server', { type: 'server', message })
}

interface FormFieldProps {
  id: string
  label: string
  error?: string
  hint?: string
  children: ReactNode
}

function FormField({ id, label, error, hint, children }: FormFieldProps) {
  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-gray-700">
        {label}
      </label>
      {children}
      {error ? (
        <p className="mt-1 text-xs text-red-600" role="alert">
          {/* Messages are fragments shared with the backend, e.g. "is required". */}
          {label} {error}
        </p>
      ) : (
        hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>
      )}
    </div>
  )
}
