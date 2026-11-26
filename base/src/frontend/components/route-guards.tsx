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

  if (user?.role !== "admin") return <Navigate to="/fazendas" replace />

  return <Outlet />
}

/** Telas de entrar e cadastro: quem já está logado vai direto para as fazendas. */
export function PublicOnly() {
  const { user, loading } = useAuth()

  if (loading) return <Loading />

  if (user) return <Navigate to="/fazendas" replace />

  return <Outlet />
}
