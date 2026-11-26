import { Navigate, Outlet, useLocation } from "react-router"

import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"

function Loading() {
  return (
    <div className="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-4 p-6" aria-busy="true">
      <Skeleton className="size-16 rounded-full" />
      <Skeleton className="h-6 w-2/3" />
      <Skeleton className="h-4 w-full" />
    </div>
  )
}

/** Só deixa passar quem está logado; senão manda para a tela de entrar, lembrando de onde veio. */
export function RequireAuth() {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) return <Loading />

  if (!user) return <Navigate to="/entrar" replace state={{ from: location.pathname }} />

  return <Outlet />
}

/** Só admin: usuário comum volta para as fazendas. */
export function RequireAdmin() {
  const { user } = useAuth()

  if (user?.role !== "admin") return <Navigate to="/" replace />

  return <Outlet />
}

/**
 * Telas de entrar e cadastro: quem já está logado sai delas. Para onde vai: o `next` ou o `from` que a tela
 * deixou no estado da navegação (ex.: a etapa da foto depois do cadastro, ou a página que pediu o login), senão as fazendas.
 *
 * É este guarda que faz a navegação logo depois do login: se a tela também chamasse navigate(), as duas
 * disputariam e a última a rodar ganharia (a pessoa acabava em /fazendas em vez de /foto).
 */
export function PublicOnly() {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) return <Loading />

  if (user) {
    const state = location.state as { next?: string; from?: string } | null

    return <Navigate to={state?.next ?? state?.from ?? "/"} replace />
  }

  return <Outlet />
}
