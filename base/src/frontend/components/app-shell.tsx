import { FlaskConicalIcon, HouseIcon, LogOutIcon, PanelLeftCloseIcon, PanelLeftOpenIcon, TractorIcon, UserRoundIcon, UsersIcon, type LucideIcon } from "lucide-react"
import { useState } from "react"
import { NavLink, Outlet, useLocation, useNavigate } from "react-router"

import { Brand } from "@/components/brand"
import { ConfirmDelete } from "@/components/confirm-delete"
import { ThemeToggle } from "@/components/theme-toggle"
import { UserAvatar } from "@/components/user-avatar"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useAuth } from "@/lib/auth"
import { notify } from "@/lib/notify"
import { cn } from "@/lib/utils"

interface Item {
  to: string
  label: string
  icon: LucideIcon
  /** O início é "/": sem `end`, ele ficaria marcado em todas as páginas. */
  end?: boolean
  /** A cor do ícone (uma de cada tom da paleta). */
  tone: string
}

const SIDEBAR_KEY = "analise-leite:sidebar-closed"

/** O menu lateral (computador) pode ser fechado; a escolha fica guardada neste aparelho. */
function useSidebar() {
  const [closed, setClosed] = useState(() => {
    try {
      return localStorage.getItem(SIDEBAR_KEY) === "1"
    } catch {
      return false
    }
  })

  function toggle() {
    setClosed((current) => {
      try {
        localStorage.setItem(SIDEBAR_KEY, current ? "0" : "1")
      } catch {
        // sem armazenamento (janela privada): só não lembra na próxima visita
      }

      return !current
    })
  }

  return { closed, toggle }
}

function useItems(): Item[] {
  const { user } = useAuth()

  return [
    { to: "/", label: "Início", icon: HouseIcon, end: true, tone: "text-primary" },
    { to: "/fazendas", label: "Fazendas", icon: TractorIcon, tone: "text-chart-3" },
    { to: "/analises", label: "Análises", icon: FlaskConicalIcon, tone: "text-chart-5" },
    ...(user?.role === "admin" ? [{ to: "/usuarios", label: "Usuários", icon: UsersIcon, tone: "text-chart-4" }] : []),
    { to: "/perfil", label: "Perfil", icon: UserRoundIcon, tone: "text-accent-foreground" },
  ]
}

/**
 * Estrutura das telas logadas. No celular: barra de cima enxuta e menu de abas fixo embaixo (ao alcance do polegar).
 * No computador: menu lateral. O conteúdo da rota entra no <Outlet />.
 */
export function AppShell() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const items = useItems()
  const location = useLocation()
  const sidebar = useSidebar()

  async function leave() {
    try {
      await logout()
      navigate("/entrar", { replace: true })
    } catch (error) {
      notify.error("Não foi possível sair", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <div className="min-h-dvh">
      {/* computador: menu lateral */}
      <aside
        inert={sidebar.closed}
        className={cn(
          "fixed inset-y-0 left-0 hidden w-60 flex-col gap-4 border-r bg-sidebar p-4 text-sidebar-foreground shadow-xl shadow-foreground/5 backdrop-blur transition-transform duration-300 ease-out md:flex",
          sidebar.closed && "-translate-x-full",
        )}
      >
        <div className="flex items-center justify-between gap-2">
          <Brand />
          <Button variant="ghost" size="icon-sm" aria-label="Fechar o menu" title="Fechar o menu" onClick={sidebar.toggle}>
            <PanelLeftCloseIcon />
          </Button>
        </div>
        <Separator />
        <nav aria-label="Principal" className="flex flex-1 flex-col gap-1">
          {items.map(({ to, label, icon: Icon, end, tone }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors [&_svg]:size-4",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
                )
              }
            >
              <Icon className={tone} />
              {label}
            </NavLink>
          ))}
        </nav>
        <Separator />
        <div className="flex items-center gap-2.5">
          {user && <UserAvatar user={user} />}
          <div className="flex min-w-0 flex-1 flex-col leading-tight">
            <span className="truncate text-sm font-medium">{user?.name}</span>
            <span className="truncate text-xs text-muted-foreground">{user?.email}</span>
          </div>
          <ThemeToggle />
          <ConfirmDelete
            trigger={
              <Button variant="ghost" size="icon" aria-label="Sair">
                <LogOutIcon />
              </Button>
            }
            title="Sair da conta?"
            description="Você vai precisar entrar de novo para ver suas fazendas e análises."
            confirmLabel="Sair"
            variant="default"
            onConfirm={leave}
          />
        </div>
      </aside>

      {/* celular: barra de cima */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b bg-background/85 px-4 pt-[env(safe-area-inset-top)] backdrop-blur md:hidden">
        <div className="flex h-14 w-full items-center justify-between">
          <Brand size="sm" />
          <ThemeToggle />
        </div>
      </header>

      {/* computador, menu fechado: botão para abrir de novo */}
      {sidebar.closed && (
        <Button
          variant="outline"
          size="icon"
          aria-label="Abrir o menu"
          title="Abrir o menu"
          onClick={sidebar.toggle}
          className="fixed top-4 left-4 z-30 hidden bg-background/90 backdrop-blur md:inline-flex"
        >
          <PanelLeftOpenIcon />
        </Button>
      )}

      <main className={cn("transition-[padding] duration-300 ease-out", sidebar.closed ? "md:pt-12 md:pl-0" : "md:pl-60")}>
        <div key={location.pathname} className="animate-page-in mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-5 pb-28 md:px-8 md:py-8 md:pb-10">
          <Outlet />
        </div>
      </main>

      {/* celular: abas fixas embaixo, com o espaço da barra do sistema (iPhone) */}
      <nav
        aria-label="Principal"
        className="fixed inset-x-0 bottom-0 z-30 border-t bg-background/92 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
      >
        <ul className="grid" style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}>
          {items.map(({ to, label, icon: Icon, end, tone }) => (
            <li key={to}>
              <NavLink
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    "flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs font-medium transition-colors [&_svg]:size-5",
                    isActive ? "text-primary" : "text-muted-foreground",
                  )
                }
              >
                <Icon className={tone} />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </div>
  )
}
