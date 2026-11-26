import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"
import type { Analysis, AnalysisPage, Content, Farm, ModelsResponse, Sample, Stats, Tank, User } from "@/lib/types"

export const keys = {
  farms: ["farms"] as const,
  farm: (id: number) => ["farms", id] as const,
  tanks: (farmId: number) => ["farms", farmId, "tanks"] as const,
  /** Todas as análises do usuário (a página Análises e o início). */
  myAnalyses: ["analyses", "mine"] as const,
  stats: ["stats"] as const,
  analysis: (id: number) => ["analyses", id] as const,
  contents: (id: number) => ["analyses", id, "contents"] as const,
  models: ["models"] as const,
  users: ["users"] as const,
  user: (id: number) => ["users", id] as const,
  userFarms: (id: number) => ["users", id, "farms"] as const,
}

/** O início e a lista de análises mostram números de tudo: qualquer mudança em fazendas, tanques ou análises os atualiza. */
const refreshSummaries = (client: QueryClient) =>
  Promise.all([client.invalidateQueries({ queryKey: keys.stats }), client.invalidateQueries({ queryKey: keys.myAnalyses })])

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
    onSuccess: () => Promise.all([client.invalidateQueries({ queryKey: keys.farms }), refreshSummaries(client)]),
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
    onSuccess: () => Promise.all([client.invalidateQueries({ queryKey: keys.farms }), refreshSummaries(client)]),
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
    onSuccess: () => Promise.all([client.invalidateQueries({ queryKey: keys.tanks(farmId) }), refreshSummaries(client)]),
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
    onSuccess: () => Promise.all([client.invalidateQueries({ queryKey: keys.tanks(farmId) }), refreshSummaries(client)]),
  })
}

// ---- análises ----

const PAGE_SIZE = 20

/** Todas as análises do usuário (de todas as fazendas), em páginas: `fetchNextPage` carrega mais. */
export const useMyAnalyses = () =>
  useInfiniteQuery({
    queryKey: keys.myAnalyses,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => api.get<AnalysisPage>(`/analyses?limit=${PAGE_SIZE}&offset=${pageParam}`),
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((sum, page) => sum + page.items.length, 0)

      return loaded < last.total ? loaded : undefined
    },
  })

/** O resumo da página de início. Os dias seguem o fuso do aparelho. */
export const useStats = () =>
  useQuery({
    queryKey: keys.stats,
    // getTimezoneOffset é quantos minutos o fuso está ATRÁS de UTC; o servidor quer os minutos À FRENTE
    queryFn: () => api.get<Stats>(`/stats?offset=${-new Date().getTimezoneOffset()}`),
  })

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
    onSuccess: () => refreshSummaries(client),
  })
}

export function useDeleteAnalysis() {
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete(`/analyses/${id}`),
    onSuccess: () => refreshSummaries(client),
  })
}

// ---- usuários (admin) ----

export const useUsers = () => useQuery({ queryKey: keys.users, queryFn: () => api.get<User[]>("/users") })

export const useUser = (id: number) => useQuery({ queryKey: keys.user(id), queryFn: () => api.get<User>(`/users/${id}`), enabled: id > 0 })

/** As fazendas de um usuário (só o admin pode pedir as de outro). */
export const useUserFarms = (id: number) =>
  useQuery({ queryKey: keys.userFarms(id), queryFn: () => api.get<Farm[]>(`/farms?user_id=${id}`), enabled: id > 0 })

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
