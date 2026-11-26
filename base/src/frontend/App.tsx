import { lazy, Suspense } from "react"
import { Navigate, Route, Routes, useParams } from "react-router"

import { AmbientBackground } from "@/components/ambient-background"
import { AppShell } from "@/components/app-shell"
import { ListSkeleton } from "@/components/list-skeleton"
import { PublicOnly, RequireAdmin, RequireAuth } from "@/components/route-guards"

// cada tela é carregada só quando é aberta: o celular baixa menos código na primeira visita
const LoginPage = lazy(() => import("@/pages/login"))
const RegisterPage = lazy(() => import("@/pages/register"))
const HomePage = lazy(() => import("@/pages/home"))
const FarmsPage = lazy(() => import("@/pages/farms"))
const FarmDetailPage = lazy(() => import("@/pages/farm-detail"))
const NewAnalysisPage = lazy(() => import("@/pages/new-analysis"))
const AnalysesPage = lazy(() => import("@/pages/analyses"))
const AnalysisDetailPage = lazy(() => import("@/pages/analysis-detail"))
const ProfilePage = lazy(() => import("@/pages/profile"))
const UsersPage = lazy(() => import("@/pages/users"))
const UserDetailPage = lazy(() => import("@/pages/user-detail"))
const PhotoPage = lazy(() => import("@/pages/photo"))
const NotFoundPage = lazy(() => import("@/pages/not-found"))

/** O endereço antigo (a análise dentro da fazenda) leva para a nova página de análises, já com a fazenda escolhida. */
function LegacyNewAnalysis() {
  const { id } = useParams()

  return <Navigate to={`/analises/nova?fazenda=${id}`} replace />
}

export function App() {
  return (
    <>
    <AmbientBackground />
    <Suspense fallback={<div className="mx-auto w-full max-w-3xl p-4"><ListSkeleton /></div>}>
      <Routes>
        <Route element={<PublicOnly />}>
          <Route path="/entrar" element={<LoginPage />} />
          <Route path="/cadastro" element={<RegisterPage />} />
        </Route>

        <Route element={<RequireAuth />}>
          <Route path="/foto" element={<PhotoPage />} />
          <Route element={<AppShell />}>
            <Route path="/" element={<HomePage />} />
            <Route path="/fazendas" element={<FarmsPage />} />
            <Route path="/fazendas/:id" element={<FarmDetailPage />} />
            <Route path="/fazendas/:id/analises/nova" element={<LegacyNewAnalysis />} />
            <Route path="/analises" element={<AnalysesPage />} />
            <Route path="/analises/nova" element={<NewAnalysisPage />} />
            <Route path="/analises/:id" element={<AnalysisDetailPage />} />
            <Route path="/perfil" element={<ProfilePage />} />
            <Route element={<RequireAdmin />}>
              <Route path="/usuarios" element={<UsersPage />} />
              <Route path="/usuarios/:id" element={<UserDetailPage />} />
            </Route>
          </Route>
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
    </>
  )
}

export default App
