const dateTime = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" })
const dateOnly = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short", year: "numeric" })
const number = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 1 })

export const formatDateTime = (iso: string) => dateTime.format(new Date(iso))
export const formatDate = (iso: string) => dateOnly.format(new Date(iso))
export const formatLiters = (liters: number) => `${number.format(liters)} L`
export const formatNumber = (value: number) => number.format(value)

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${number.format(bytes / 1024)} KB`
  return `${number.format(bytes / (1024 * 1024))} MB`
}

/** Siglas dos estados (as mesmas que o backend aceita). */
export const STATES = [
  "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI",
  "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
] as const

/** Primeiras letras do nome, para o avatar. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  return ((parts[0]?.[0] ?? "") + (parts.length > 1 ? parts[parts.length - 1][0] : "")).toUpperCase()
}

/** Mensagens para o ?error= que o backend manda ao voltar do login com o Google. */
export const GOOGLE_ERRORS: Record<string, string> = {
  access_denied: "O login com o Google foi cancelado.",
  invalid_state: "A sessão do login expirou. Tente entrar com o Google de novo.",
  invalid_request: "O Google não devolveu os dados do login. Tente de novo.",
  exchange_failed: "Não foi possível confirmar o login com o Google. Tente de novo.",
  email_not_verified: "O email dessa conta do Google não está verificado.",
  account_exists: "Já existe uma conta com esse email e senha. Entre com o email e a senha.",
  server_error: "Algo deu errado ao entrar com o Google. Tente de novo.",
}
