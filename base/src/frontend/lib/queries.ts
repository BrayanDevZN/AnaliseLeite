import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"
import type { Analysis, Content, Farm, ModelsResponse, Sample, Tank, User } from "@/lib/types"

export const keys = {
  farms: ["farms"] as const,
  farm: (id: number) => ["farms", id] as const,
  tanks: (farmId: number) => ["farms", farmId, "tanks"] as const,
  analyses: (farmId: number) => ["farms", farmId, "analyses"] as const,
  analysis: (id: number) => ["analyses", id] as const,
  contents: (id: number) => ["analyses", id, "contents"] as const,
  models: ["models"] as const,
  users: ["users"] as const,
}

// As consultas por id só rodam com um id válido (> 0): assim uma tela que ainda não sabe o id não chama a API à toa.

// ---- fazendas ----

export const useFarms = () => useQuery({ queryKey: keys.farms, queryFn: () => api.get<Farm[]>("/farms") })

export const useFarm = (id: number) => useQuery({ queryKey: keys.farm(id), queryFn: () => api.get<Farm>(`/farms/${id}`), enabled: id > 0 })

export interface FarmInput {
  name: string
  city: string
  state: string
}

export function useCreateFarm() {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: FarmInput) => api.post<Farm>("/farms", input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.farms }),
  })
}

export function useUpdateFarm(id: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: Partial<FarmInput>) => api.patch<Farm>(`/farms/${id}`, input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.farms }),
  })
}

export function useDeleteFarm() {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete(`/farms/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.farms }),
  })
}

// ---- tanques ----

export const useTanks = (farmId: number) =>
  useQuery({ queryKey: keys.tanks(farmId), queryFn: () => api.get<Tank[]>(`/farms/${farmId}/tanks`), enabled: farmId > 0 })

export interface TankInput {
  type: string
  capacity_liters: number
}

export function useCreateTank(farmId: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: TankInput) => api.post<Tank>(`/farms/${farmId}/tanks`, input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.tanks(farmId) }),
  })
}

export function useUpdateTank(farmId: number, id: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: Partial<TankInput>) => api.patch<Tank>(`/tanks/${id}`, input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.tanks(farmId) }),
  })
}

export function useDeleteTank(farmId: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete(`/tanks/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.tanks(farmId) }),
  })
}

// ---- análises ----

export const useAnalyses = (farmId: number) =>
  useQuery({ queryKey: keys.analyses(farmId), queryFn: () => api.get<Analysis[]>(`/farms/${farmId}/analyses`), enabled: farmId > 0 })

export const useAnalysis = (id: number) =>
  useQuery({ queryKey: keys.analysis(id), queryFn: () => api.get<Analysis>(`/analyses/${id}`), enabled: id > 0 })

export const useContents = (id: number) =>
  useQuery({ queryKey: keys.contents(id), queryFn: () => api.get<Content[]>(`/analyses/${id}/contents`), enabled: id > 0 })

export const useModels = () =>
  useQuery({ queryKey: keys.models, queryFn: () => api.get<ModelsResponse>("/models"), staleTime: 5 * 60_000 })

export interface AnalysisInput {
  tankId: number | null
  /** Vazio usa o modelo padrão do pipeline. */
  model: string
  /** Ou uma amostra, ou um arquivo (csv ou parquet): nunca os dois. */
  sample?: Sample
  file?: File
}

/** Cria a análise: manda o arquivo em multipart ou a amostra em JSON. A resposta traz o resultado do modelo. */
export function useCreateAnalysis(farmId: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: AnalysisInput) => {
      const path = `/farms/${farmId}/analyses`

      if (input.file) {
        const form = new FormData()

        if (input.tankId) form.set("tank_id", String(input.tankId))
        if (input.model) form.set("model", input.model)

        form.set("file", input.file)

        return api.upload<Analysis>(path, form)
      }

      return api.post<Analysis>(path, {
        ...(input.tankId ? { tank_id: input.tankId } : {}),
        ...(input.model ? { model: input.model } : {}),
        sample: input.sample,
      })
    },
    onSuccess: () => client.invalidateQueries({ queryKey: keys.analyses(farmId) }),
  })
}

export function useDeleteAnalysis(farmId: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete(`/analyses/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.analyses(farmId) }),
  })
}

// ---- usuários (admin) ----

export const useUsers = () => useQuery({ queryKey: keys.users, queryFn: () => api.get<User[]>("/users") })

export interface UserInput {
  name: string
  email: string
  password?: string
  role: string
}

export function useCreateUser() {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: UserInput) => api.post<User>("/users", input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.users }),
  })
}

export function useUpdateUser(id: number) {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: Partial<UserInput>) => api.patch<User>(`/users/${id}`, input),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.users }),
  })
}

export function useDeleteUser() {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete(`/users/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.users }),
  })
}
