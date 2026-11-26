import { lazy, Suspense } from "react"
import { Navigate, Route, Routes } from "react-router"

import { AppShell } from "@/components/app-shell"
import { ListSkeleton } from "@/components/list-skeleton"
import { PublicOnly, RequireAdmin, RequireAuth } from "@/components/route-guards"

// cada tela é carregada só quando é aberta: o celular baixa menos código na primeira visita
const LoginPage = lazy(() => import("@/pages/login"))
const RegisterPage = lazy(() => import("@/pages/register"))
const FarmsPage = lazy(() => import("@/pages/farms"))
const FarmDetailPage = lazy(() => import("@/pages/farm-detail"))
const NewAnalysisPage = lazy(() => import("@/pages/new-analysis"))
const AnalysisDetailPage = lazy(() => import("@/pages/analysis-detail"))
const ProfilePage = lazy(() => import("@/pages/profile"))
const UsersPage = lazy(() => import("@/pages/users"))
const NotFoundPage = lazy(() => import("@/pages/not-found"))

export function App() {
  return (
    <Suspense fallback={<div className="mx-auto w-full max-w-3xl p-4"><ListSkeleton /></div>}>
      <Routes>
        <Route element={<PublicOnly />}>
          <Route path="/entrar" element={<LoginPage />} />
          <Route path="/cadastro" element={<RegisterPage />} />
        </Route>

        <Route element={<RequireAuth />}>
          <Route element={<AppShell />}>
            <Route path="/" element={<Navigate to="/fazendas" replace />} />
            <Route path="/fazendas" element={<FarmsPage />} />
            <Route path="/fazendas/:id" element={<FarmDetailPage />} />
            <Route path="/fazendas/:id/analises/nova" element={<NewAnalysisPage />} />
            <Route path="/analises/:id" element={<AnalysisDetailPage />} />
            <Route path="/perfil" element={<ProfilePage />} />
            <Route element={<RequireAdmin />}>
              <Route path="/usuarios" element={<UsersPage />} />
            </Route>
          </Route>
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  )
}

export default App
