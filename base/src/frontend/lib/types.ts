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
  tank_id: number | null
  created_at: string
  content?: Content
  /** "aprovado" ou "reprovado": só vem na resposta de criação (o banco não guarda). */
  result?: string
  /** Uma linha por linha do arquivo enviado: só vem na resposta de criação. */
  rows?: Record<string, unknown>[]
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
