import { z } from 'zod'
import { charCount } from './entity'

// Mirrors the backend rules in backend/internal/validation (RegisterInput and
// LoginInput). Messages match the backend so client and server errors read the same.

export const EMAIL_MAX = 254
export const PASSWORD_MIN = 8
/** bcrypt only uses the first 72 bytes, so the backend limits bytes, not characters. */
export const PASSWORD_MAX_BYTES = 72

const byteLength = (s: string) => new TextEncoder().encode(s).length

const required = { error: 'is required', abort: true } as const

/** Trimmed and lowercased, like the backend, so "Budi@Example.com " is the same account. */
const normalizedEmail = () => z.string().trim().toLowerCase().min(1, required)

export const loginSchema = z.object({
  email: normalizedEmail(),
  // Only presence is checked: a short password is a wrong password (401).
  password: z.string().min(1, required),
})

export const registerSchema = z.object({
  // The backend checks length before format, so only one error shows at a time.
  email: normalizedEmail()
    .refine((s) => charCount(s) <= EMAIL_MAX, { error: `must be at most ${EMAIL_MAX} characters`, abort: true })
    .refine((s) => z.email().safeParse(s).success, 'must be a valid email address'),
  // Not trimmed: spaces are part of the password.
  password: z
    .string()
    .min(1, required)
    .refine((s) => charCount(s) >= PASSWORD_MIN, {
      error: `must be at least ${PASSWORD_MIN} characters`,
      abort: true,
    })
    .refine((s) => byteLength(s) <= PASSWORD_MAX_BYTES, `must be at most ${PASSWORD_MAX_BYTES} bytes`),
})

/** Both forms have the same fields. */
export type CredentialsFormValues = z.input<typeof loginSchema>
