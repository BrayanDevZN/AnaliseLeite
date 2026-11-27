// Tipos que espelham o schema do backend (base/src/backend/schema).

export type Role = "admin" | "user"

export interface User {
  id: number
  name: string
  email: string
  role: Role
  has_password: boolean
  has_profile_photo: boolean
  created_at: string
}

export interface Session {
  expires_in: number
  user: User
}

export interface Farm {
  id: number
  user_id: number
  name: string
  city: string
  state: string
  has_photo: boolean
  created_at: string
}

export interface Tank {
  id: number
  farm_id: number
  type: string
  capacity_liters: number
  created_at: string
}

export interface Content {
  id: number
  analysis_id: number
  kind: "content" | "file"
  content?: Sample
  file_size?: number
  created_at: string
}

/** Uma amostra de leite. Os nomes dos campos são os do pipeline. */
export interface Sample {
  temperature?: number
  state?: string
  data?: string
  ccs?: number
  cpp?: number
  cidade?: string
  tanque?: string
}

export interface Analysis {
  id: number
  farm_id: number
  /** O nome da fazenda: só vem na lista de todas as análises (GET /analyses). */
  farm_name?: string
  tank_id: number | null
  created_at: string
  /** "sample" (uma amostra), "file" (um arquivo) ou "image" (uma imagem lida pelo agente). */
  kind: "sample" | "file" | "image"
  /** O modelo usado; vazio é o modelo padrão. */
  model: string
  /** Quantas amostras o modelo aprovou e reprovou. Os dois zero: o resultado não foi guardado (análise antiga). */
  approved: number
  rejected: number
  content?: Content
  /** "aprovado" ou "reprovado": só vem na resposta de criação (o banco não guarda). */
  result?: string
  /** Uma linha por linha do arquivo enviado: só vem na resposta de criação. */
  rows?: Record<string, unknown>[]
  /** Análise em texto do agente de imagem, quando a imagem é sobre leite mas não tem números: só vem na resposta de criação. */
  text?: string
}

/** Uma página da lista de análises. */
export interface AnalysisPage {
  items: Analysis[]
  total: number
}

export interface DayStats {
  /** O dia, como 2025-09-01. */
  date: string
  analyses: number
  approved: number
  rejected: number
}

/** O resumo da página de início. */
export interface Stats {
  farms: number
  tanks: number
  analyses: number
  approved: number
  rejected: number
  /** Os últimos 7 dias, do mais antigo para hoje. */
  days: DayStats[]
}

export interface ModelsResponse {
  models: string[]
}

export interface FieldError {
  field: string
  message: string
}

export interface ErrorBody {
  detail: string
  errors?: FieldError[]
}
