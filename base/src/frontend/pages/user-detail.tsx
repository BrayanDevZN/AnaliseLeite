import { MapPinIcon, MilkIcon, ShieldCheckIcon, ShieldOffIcon, Trash2Icon } from "lucide-react"
import { Link, Navigate, useNavigate, useParams } from "react-router"

import { ConfirmDelete } from "@/components/confirm-delete"
import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { FarmThumb } from "@/components/farm-thumb"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { UserAvatar } from "@/components/user-avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { formatDate, formatLiters } from "@/lib/format"
import { notify } from "@/lib/notify"
import { useDeleteUser, useTanks, useUpdateUser, useUser, useUserFarms } from "@/lib/queries"
import type { Farm } from "@/lib/types"

/** Uma fazenda do usuário, com os tanques dela. */
function FarmCard({ farm }: { farm: Farm }) {
  const tanks = useTanks(farm.id)

  return (
    <Card size="sm">
      <CardHeader className="flex flex-row items-center gap-3">
        <FarmThumb farm={farm} className="size-12" />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <CardTitle className="truncate">
            <Link to={`/fazendas/${farm.id}`} className="underline-offset-4 hover:underline">
              {farm.name}
            </Link>
          </CardTitle>
          <CardDescription className="flex items-center gap-1">
            <MapPinIcon className="size-3.5" />
            {farm.city} - {farm.state}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <Separator />
        {tanks.isPending && <Skeleton className="h-6 w-1/2" />}
        {tanks.isError && <ErrorAlert error={tanks.error} title="Não foi possível carregar os tanques" />}
        {tanks.data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum tanque cadastrado.</p>}
        {tanks.data && tanks.data.length > 0 && (
          <ul className="flex flex-col gap-2">
            {tanks.data.map((tank) => (
              <li key={tank.id} className="flex items-center justify-between gap-3 text-sm">
                <span className="flex min-w-0 items-center gap-2">
                  <MilkIcon className="size-4 shrink-0 text-primary" />
                  <span className="truncate capitalize">{tank.type}</span>
                </span>
                <Badge variant="secondary">{formatLiters(tank.capacity_liters)}</Badge>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

export default function UserDetailPage() {
  const params = useParams()
  const id = Number(params.id)
  const navigate = useNavigate()
  const { user: me } = useAuth()

  const user = useUser(id)
  const farms = useUserFarms(id)
  const update = useUpdateUser(id)
  const remove = useDeleteUser()

  if (!Number.isInteger(id) || id <= 0 || (user.error instanceof ApiError && user.error.status === 404)) {
    return <Navigate to="/usuarios" replace />
  }

  if (user.isError) return <ErrorAlert error={user.error} title="Não foi possível carregar o usuário" />

  if (user.isPending) {
    return (
      <>
        <Skeleton className="h-8 w-2/3" />
        <ListSkeleton />
      </>
    )
  }

  const item = user.data
  const isAdmin = item.role === "admin"
  const isMe = item.id === me?.id

  async function toggleAdmin() {
    try {
      await update.mutateAsync({ role: isAdmin ? "user" : "admin" })
      notify.success(isAdmin ? "Administrador removido" : "Agora é administrador")
    } catch (error) {
      notify.error("Não foi possível mudar o papel", error instanceof Error ? error.message : undefined)
    }
  }

  async function removeUser() {
    try {
      await remove.mutateAsync(id)
      notify.success("Usuário apagado")
      navigate("/usuarios", { replace: true })
    } catch (error) {
      notify.error("Não foi possível apagar", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <>
      <PageHeader
        title={item.name}
        description="Fazendas e tanques desta pessoa."
        back="/usuarios"
        actions={
          !isMe && (
            <ConfirmDelete
              trigger={
                <Button variant="outline" size="icon" aria-label={`Apagar ${item.name}`}>
                  <Trash2Icon />
                </Button>
              }
              title={`Apagar ${item.name}?`}
              description="A pessoa perde o acesso na hora, e as fazendas, tanques e análises dela também são apagados."
              onConfirm={removeUser}
            />
          )
        }
      />

      <Card>
        <CardHeader className="flex flex-row items-center gap-3">
          <UserAvatar user={item} />
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CardTitle className="truncate">{item.email}</CardTitle>
            <CardDescription className="flex flex-wrap items-center gap-2">
              <Badge variant={isAdmin ? "default" : "secondary"}>
                {isAdmin && <ShieldCheckIcon data-icon="inline-start" />}
                {isAdmin ? "Administrador" : "Usuário"}
              </Badge>
              {!item.has_password && <Badge variant="outline">Só Google</Badge>}
              <span>Desde {formatDate(item.created_at)}</span>
            </CardDescription>
          </div>
        </CardHeader>
        {!isMe && (
          <CardContent>
            <Button variant={isAdmin ? "outline" : "default"} onClick={toggleAdmin} disabled={update.isPending}>
              {update.isPending ? <Spinner data-icon="inline-start" /> : isAdmin ? <ShieldOffIcon data-icon="inline-start" /> : <ShieldCheckIcon data-icon="inline-start" />}
              {isAdmin ? "Remover administrador" : "Tornar administrador"}
            </Button>
          </CardContent>
        )}
      </Card>

      <section className="flex flex-col gap-3" aria-labelledby="farms-title">
        <h2 id="farms-title" className="font-heading text-lg font-semibold">
          Fazendas
        </h2>

        {farms.isPending && <ListSkeleton count={2} />}
        {farms.isError && <ErrorAlert error={farms.error} title="Não foi possível carregar as fazendas" />}
        {farms.data?.length === 0 && <EmptyState title="Nenhuma fazenda" description="Esta pessoa ainda não cadastrou fazendas." />}

        <ul className="flex flex-col gap-3">
          {farms.data?.map((farm) => (
            <li key={farm.id}>
              <FarmCard farm={farm} />
            </li>
          ))}
        </ul>
      </section>
    </>
  )
}
