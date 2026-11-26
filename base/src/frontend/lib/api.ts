import type { ErrorBody } from "@/lib/types"

/** URL do backend. Em desenvolvimento é a local; em produção defina VITE_API_URL. */
export const API_URL: string = (import.meta.env.VITE_API_URL ?? "http://localhost:8080").replace(/\/$/, "")

/** Para onde mandar o navegador para entrar com o Google (o backend redireciona para o Google). */
export const GOOGLE_LOGIN_URL = `${API_URL}/auth/google`

/** Erro de uma chamada à API, já com mensagem em português para mostrar ao usuário. */
export class ApiError extends Error {
  status: number
  /** Erros por campo (422 do backend), indexados pelo nome do campo. */
  fields: Record<string, string>
  retryAfter?: number

  constructor(status: number, message: string, fields: Record<string, string> = {}, retryAfter?: number) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.fields = fields
    this.retryAfter = retryAfter
  }
}

const MESSAGES: Record<number, string> = {
  0: "Não foi possível falar com o servidor. Verifique sua conexão.",
  401: "Sua sessão expirou. Entre novamente.",
  403: "Você não tem permissão para fazer isso.",
  404: "Não encontramos o que você procurava.",
  409: "Já existe um registro com esses dados.",
  429: "Muitas tentativas em pouco tempo. Aguarde um instante e tente de novo.",
  502: "O serviço do modelo recusou a análise. Confira os dados e tente de novo.",
  503: "Esse recurso está indisponível no momento.",
}

/** O backend responde as mensagens de validação em inglês; aqui elas viram português. */
const FIELD_MESSAGES: Record<string, string> = {
  "is required": "Campo obrigatório.",
  "is too long": "Texto longo demais.",
  "is not a valid email": "Digite um email válido.",
  "must have at least 8 characters": "Use pelo menos 8 caracteres.",
  "must have at most 72 bytes": "A senha é longa demais (máximo de 72 caracteres).",
  "must be admin or user": "Escolha administrador ou usuário.",
  "must be a brazilian state abbreviation, like MG": "Escolha um estado válido, como MG.",
  "is out of the valid range": "Valor fora do intervalo aceito.",
  "must be greater than zero": "Use um número maior que zero.",
  "must be a positive number": "Use um número maior que zero.",
  "must be a number": "Digite um número.",
  "is not a valid model name": "Nome de modelo inválido.",
  "must be a date like 2025-09-01": "Use uma data válida.",
  "must be a .csv or .parquet file": "Envie um arquivo .csv ou .parquet.",
  "is empty": "O arquivo está vazio.",
  "is too big": "Arquivo grande demais (máximo de 20 MB).",
  "must have at least one field": "Preencha pelo menos um campo.",
  "send either a sample or a file, not both": "Envie uma amostra ou um arquivo, não os dois.",
  "send a sample or a file": "Envie uma amostra ou um arquivo.",
  "is not a tank of this farm": "Esse tanque não é desta fazenda.",
  "is not a valid upload or is too big": "O envio é inválido ou grande demais.",
  "could not be read": "Não foi possível ler o arquivo.",
  "is not valid JSON for this route": "Os dados enviados não são válidos.",
}

async function toError(response: Response): Promise<ApiError> {
  let body: Partial<ErrorBody> = {}

  try {
    body = await response.json()
  } catch {
    // corpo vazio ou que não é JSON
  }

  const fields: Record<string, string> = {}

  for (const item of body.errors ?? []) {
    fields[item.field] = FIELD_MESSAGES[item.message] ?? item.message
  }

  const retry = Number(response.headers.get("Retry-After")) || undefined
  const message =
    response.status === 422
      ? "Confira os campos destacados."
      : (MESSAGES[response.status] ?? "Algo deu errado. Tente novamente.")

  return new ApiError(response.status, message, fields, retry)
}

// Rotas em que um 401 significa "credenciais erradas", e não "token vencido": nelas não adianta renovar.
const NO_REFRESH = ["/auth/login", "/auth/register", "/auth/refresh", "/auth/logout"]

let refreshing: Promise<boolean> | null = null

/** Troca o refresh token (cookie) por tokens novos. Várias chamadas ao mesmo tempo dividem a mesma renovação. */
function refreshSession(): Promise<boolean> {
  refreshing ??= fetch(`${API_URL}/auth/refresh`, { method: "POST", credentials: "include" })
    .then((response) => response.ok)
    .catch(() => false)
    .finally(() => {
      refreshing = null
    })

  return refreshing
}

interface Options {
  method?: "GET" | "POST" | "PATCH" | "DELETE"
  body?: unknown
  form?: FormData
}

/** Faz a chamada com os cookies de sessão. Se o token vencer (401), renova uma vez e repete. */
async function request<T>(path: string, options: Options = {}, retried = false): Promise<T> {
  const headers: Record<string, string> = {}
  let body: BodyInit | undefined

  if (options.form) {
    body = options.form // o navegador define o Content-Type do multipart, com o boundary
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json"
    body = JSON.stringify(options.body)
  }

  let response: Response

  try {
    response = await fetch(`${API_URL}${path}`, { method: options.method ?? "GET", headers, body, credentials: "include" })
  } catch {
    throw new ApiError(0, MESSAGES[0])
  }

  if (response.status === 401 && !retried && !NO_REFRESH.includes(path)) {
    if (await refreshSession()) {
      return request<T>(path, options, true)
    }
  }

  if (!response.ok) {
    throw await toError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: "POST", body }),
  patch: <T>(path: string, body: unknown) => request<T>(path, { method: "PATCH", body }),
  delete: (path: string) => request<void>(path, { method: "DELETE" }),
  upload: <T>(path: string, form: FormData) => request<T>(path, { method: "POST", form }),
}
