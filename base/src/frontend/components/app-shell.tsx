import { LogOutIcon, TractorIcon, UserRoundIcon, UsersIcon, type LucideIcon } from "lucide-react"
import { NavLink, Outlet, useNavigate } from "react-router"

import { Brand } from "@/components/brand"
import { ThemeToggle } from "@/components/theme-toggle"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useAuth } from "@/lib/auth"
import { initials } from "@/lib/format"
import { cn } from "@/lib/utils"

interface Item {
  to: string
  label: string
  icon: LucideIcon
}

function useItems(): Item[] {
  const { user } = useAuth()

  return [
    { to: "/fazendas", label: "Fazendas", icon: TractorIcon },
    ...(user?.role === "admin" ? [{ to: "/usuarios", label: "Usuários", icon: UsersIcon }] : []),
    { to: "/perfil", label: "Perfil", icon: UserRoundIcon },
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

  async function leave() {
    await logout()
    navigate("/entrar", { replace: true })
  }

  return (
    <div className="min-h-dvh">
      {/* computador: menu lateral */}
      <aside className="fixed inset-y-0 left-0 hidden w-60 flex-col gap-4 border-r bg-sidebar p-4 text-sidebar-foreground md:flex">
        <Brand />
        <Separator />
        <nav aria-label="Principal" className="flex flex-1 flex-col gap-1">
          {items.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors [&_svg]:size-4",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
                )
              }
            >
              <Icon />
              {label}
            </NavLink>
          ))}
        </nav>
        <Separator />
        <div className="flex items-center gap-2.5">
          <Avatar>
            <AvatarFallback>{initials(user?.name ?? "")}</AvatarFallback>
          </Avatar>
          <div className="flex min-w-0 flex-1 flex-col leading-tight">
            <span className="truncate text-sm font-medium">{user?.name}</span>
            <span className="truncate text-xs text-muted-foreground">{user?.email}</span>
          </div>
          <ThemeToggle />
          <Button variant="ghost" size="icon" aria-label="Sair" onClick={leave}>
            <LogOutIcon />
          </Button>
        </div>
      </aside>

      {/* celular: barra de cima */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b bg-background/85 px-4 pt-[env(safe-area-inset-top)] backdrop-blur md:hidden">
        <div className="flex h-14 w-full items-center justify-between">
          <Brand size="sm" />
          <ThemeToggle />
        </div>
      </header>

      <main className="md:pl-60">
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-5 pb-28 md:px-8 md:py-8 md:pb-10">
          <Outlet />
        </div>
      </main>

      {/* celular: abas fixas embaixo, com o espaço da barra do sistema (iPhone) */}
      <nav
        aria-label="Principal"
        className="fixed inset-x-0 bottom-0 z-30 border-t bg-background/92 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
      >
        <ul className="grid" style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}>
          {items.map(({ to, label, icon: Icon }) => (
            <li key={to}>
              <NavLink
                to={to}
                className={({ isActive }) =>
                  cn(
                    "flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs font-medium transition-colors [&_svg]:size-5",
                    isActive ? "text-primary" : "text-muted-foreground",
                  )
                }
              >
                <Icon />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </div>
  )
}
