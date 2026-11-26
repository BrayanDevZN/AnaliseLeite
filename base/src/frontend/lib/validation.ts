/** Regras de senha ao CRIAR ou TROCAR a senha. O backend confere as mesmas (e é ele quem vale). */
export const PASSWORD_RULES = [
  { id: "length", label: "Pelo menos 8 caracteres", ok: (password: string) => [...password].length >= 8 },
  { id: "upper", label: "Uma letra maiúscula", ok: (password: string) => /\p{Lu}/u.test(password) },
  { id: "lower", label: "Uma letra minúscula", ok: (password: string) => /\p{Ll}/u.test(password) },
  { id: "digit", label: "Um número", ok: (password: string) => /\p{Nd}/u.test(password) },
] as const

/** O servidor só lê os 72 primeiros bytes de uma senha. */
const MAX_PASSWORD_BYTES = 72

/** Devolve o que está errado com a senha, ou null se ela serve. */
export function passwordProblem(password: string): string | null {
  if (new TextEncoder().encode(password).length > MAX_PASSWORD_BYTES) {
    return "A senha é longa demais (máximo de 72 caracteres)."
  }

  if (!PASSWORD_RULES.every((rule) => rule.ok(password))) {
    return "A senha precisa seguir todas as regras abaixo."
  }

  return null
}

export const GMAIL_MESSAGE = "Use um email @gmail.com."

/** Só emails @gmail.com podem criar conta. */
export const isGmail = (email: string) => /^[^\s@]+@gmail\.com$/i.test(email.trim())
