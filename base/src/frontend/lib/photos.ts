import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { blobToDataUrl } from "@/lib/image"
import { keys } from "@/lib/queries"
import type { Farm, User } from "@/lib/types"

/**
 * Carrega uma foto do backend (com os cookies de sessão) e devolve como "data:..." para usar em <img>.
 * Um <img src="http://api/..."> não serviria: o navegador não manda o cookie de login em imagens de outro site.
 * Com path nulo (ou sem foto), devolve null.
 */
export function usePhotoUrl(path: string | null, version = 0) {
  return useQuery({
    queryKey: ["photo", path, version],
    enabled: path !== null,
    staleTime: 10 * 60_000,
    retry: false,
    queryFn: async () => {
      try {
        return await blobToDataUrl(await api.blob(path!))
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null

        throw error
      }
    },
  }).data ?? null
}

/** Foto de perfil de um usuário: o endereço só existe se ele tem foto. */
export const useUserPhoto = (user: Pick<User, "id" | "has_profile_photo"> | null | undefined) =>
  usePhotoUrl(user?.has_profile_photo ? `/users/${user.id}/photo` : null)

/** Foto de uma fazenda. */
export const useFarmPhoto = (farm: Pick<Farm, "id" | "has_photo"> | null | undefined) =>
  usePhotoUrl(farm?.has_photo ? `/farms/${farm.id}/photo` : null)

/** Troca ou remove a foto de perfil de quem está logado. */
export function useMyPhoto() {
  const client = useQueryClient()
  const { setUser } = useAuth()

  const done = (user: User) => {
    setUser(user)
    client.invalidateQueries({ queryKey: ["photo"] })
    client.invalidateQueries({ queryKey: keys.users })
  }

  return {
    set: useMutation({ mutationFn: (image: Blob) => api.putImage<User>("/me/photo", image), onSuccess: done }),
    remove: useMutation({ mutationFn: () => api.deleteJson<User>("/me/photo"), onSuccess: done }),
  }
}

/** Troca ou remove a foto de uma fazenda. */
export function useFarmPhotoMutations(farmId: number) {
  const client = useQueryClient()

  const done = () => {
    client.invalidateQueries({ queryKey: ["photo"] })
    client.invalidateQueries({ queryKey: keys.farms })
  }

  return {
    set: useMutation({ mutationFn: (image: Blob) => api.putImage<Farm>(`/farms/${farmId}/photo`, image), onSuccess: done }),
    remove: useMutation({ mutationFn: () => api.deleteJson<Farm>(`/farms/${farmId}/photo`), onSuccess: done }),
  }
}
