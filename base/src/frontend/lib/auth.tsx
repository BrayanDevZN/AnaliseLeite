import { createContext, useContext, useMemo, type ReactNode } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"

import { api, ApiError } from "@/lib/api"
import type { Session, User } from "@/lib/types"

interface AuthContextValue {
  /** O usuário logado, ou null se não há login. */
  user: User | null
  /** true enquanto ainda não sabemos se há login (primeira carga da página). */
  loading: boolean
  login: (email: string, password: string) => Promise<User>
  register: (name: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  /** Troca o usuário em memória depois de editar o perfil. */
  setUser: (user: User) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

const ME = ["me"] as const

export function AuthProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient()

  const me = useQuery({
    queryKey: ME,
    // 401 não é erro aqui: só quer dizer que ninguém está logado
    queryFn: async () => {
      try {
        return await api.get<User>("/me")
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) return null

        throw error
      }
    },
    staleTime: 60_000,
    retry: false,
  })

  const value = useMemo<AuthContextValue>(
    () => ({
      user: me.data ?? null,
      loading: me.isPending,
      async login(email, password) {
        const session = await api.post<Session>("/auth/login", { email, password })

        client.setQueryData(ME, session.user)

        return session.user
      },
      async register(name, email, password) {
        await api.post<User>("/auth/register", { name, email, password })
      },
      async logout() {
        // Se o servidor não confirmar, o erro sobe para quem chamou: os cookies são HttpOnly (o navegador só os
        // apaga quando o servidor manda), então fingir que saiu deixaria a pessoa logada de verdade.
        await api.post("/auth/logout")

        // Atualiza a consulta de "quem está logado" (a que a tela observa) e apaga todo o resto,
        // para os dados de um usuário nunca aparecerem para o próximo. Não use client.clear(): ele
        // joga fora a consulta que a tela observa, e a tela continuaria achando que há usuário.
        client.setQueryData(ME, null)
        client.removeQueries({ predicate: (query) => query.queryKey[0] !== ME[0] })
      },
      setUser: (user) => client.setQueryData(ME, user),
    }),
    [me.data, me.isPending, client],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)

  if (!value) throw new Error("useAuth precisa estar dentro do AuthProvider")

  return value
}
