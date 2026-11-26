import { useState } from "react"
import { MilkIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { Navigate, useNavigate, useParams } from "react-router"

import { ConfirmDelete } from "@/components/confirm-delete"
import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { FarmFormDrawer } from "@/components/farm-form"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { TankFormDrawer } from "@/components/tank-form"
import { useFarmPhoto } from "@/lib/photos"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ApiError } from "@/lib/api"
import { formatLiters } from "@/lib/format"
import { notify } from "@/lib/notify"
import { useDeleteFarm, useDeleteTank, useFarm, useTanks } from "@/lib/queries"
import type { Tank } from "@/lib/types"

export default function FarmDetailPage() {
  const params = useParams()
  const id = Number(params.id)
  const navigate = useNavigate()

  const farm = useFarm(id)
  const tanks = useTanks(id)
  const deleteFarm = useDeleteFarm()
  const deleteTank = useDeleteTank(id)
  const cover = useFarmPhoto(farm.data)

  const [editingFarm, setEditingFarm] = useState(false)
  const [tankDrawer, setTankDrawer] = useState<{ open: boolean; tank?: Tank }>({ open: false })

  if (!Number.isInteger(id) || id <= 0 || (farm.error instanceof ApiError && farm.error.status === 404)) {
    return <Navigate to="/fazendas" replace />
  }

  if (farm.isError) return <ErrorAlert error={farm.error} title="Não foi possível carregar a fazenda" />

  if (farm.isPending) {
    return (
      <>
        <Skeleton className="h-8 w-2/3" />
        <ListSkeleton />
      </>
    )
  }

  async function removeFarm() {
    try {
      await deleteFarm.mutateAsync(id)
      notify.success("Fazenda apagada")
      navigate("/fazendas", { replace: true })
    } catch (error) {
      notify.error("Não foi possível apagar", error instanceof Error ? error.message : undefined)
    }
  }

  async function removeTank(tank: Tank) {
    try {
      await deleteTank.mutateAsync(tank.id)
      notify.success("Tanque apagado")
    } catch (error) {
      notify.error("Não foi possível apagar", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <>
      <PageHeader
        title={farm.data.name}
        description={`${farm.data.city} - ${farm.data.state}`}
        back="/fazendas"
        actions={
          <>
            <Button variant="outline" size="icon" aria-label="Editar fazenda" onClick={() => setEditingFarm(true)}>
              <PencilIcon />
            </Button>
            <ConfirmDelete
              trigger={
                <Button variant="outline" size="icon" aria-label="Apagar fazenda">
                  <Trash2Icon />
                </Button>
              }
              title="Apagar esta fazenda?"
              description="A fazenda sai da lista. Apague antes os tanques e as análises dela se não quiser deixá-los soltos."
              onConfirm={removeFarm}
            />
          </>
        }
      />

      {cover && <img src={cover} alt={`Foto da fazenda ${farm.data.name}`} className="aspect-video w-full rounded-xl border object-cover" />}

      <section className="flex flex-col gap-3" aria-labelledby="tanks-title">
        <div className="flex items-center justify-between gap-3">
          <h2 id="tanks-title" className="font-heading text-lg font-semibold">
            Tanques
          </h2>
          <Button variant="outline" size="sm" onClick={() => setTankDrawer({ open: true })}>
            <PlusIcon data-icon="inline-start" />
            Novo tanque
          </Button>
        </div>

        {tanks.isPending && <ListSkeleton count={2} />}
        {tanks.isError && <ErrorAlert error={tanks.error} title="Não foi possível carregar os tanques" />}
        {tanks.data?.length === 0 && <EmptyState title="Nenhum tanque" description="Cadastre o tanque onde o leite desta fazenda é guardado." />}

        <ul className="flex flex-col gap-3">
          {tanks.data?.map((tank) => (
            <li key={tank.id}>
              <Card size="sm">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <MilkIcon className="size-4 text-primary" />
                    <span className="truncate capitalize">{tank.type}</span>
                  </CardTitle>
                  <CardDescription>Capacidade: {formatLiters(tank.capacity_liters)}</CardDescription>
                  <CardAction className="flex gap-1">
                    <Button variant="ghost" size="icon-sm" aria-label={`Editar o tanque ${tank.type}`} onClick={() => setTankDrawer({ open: true, tank })}>
                      <PencilIcon />
                    </Button>
                    <ConfirmDelete
                      trigger={
                        <Button variant="ghost" size="icon-sm" aria-label={`Apagar o tanque ${tank.type}`}>
                          <Trash2Icon />
                        </Button>
                      }
                      title="Apagar este tanque?"
                      description="As análises que usaram este tanque continuam, mas deixam de apontar para ele."
                      onConfirm={() => removeTank(tank)}
                    />
                  </CardAction>
                </CardHeader>
              </Card>
            </li>
          ))}
        </ul>
      </section>

      <FarmFormDrawer open={editingFarm} onOpenChange={setEditingFarm} farm={farm.data} />
      <TankFormDrawer open={tankDrawer.open} onOpenChange={(open) => setTankDrawer((state) => ({ ...state, open }))} farmId={id} tank={tankDrawer.tank} />
    </>
  )
}
