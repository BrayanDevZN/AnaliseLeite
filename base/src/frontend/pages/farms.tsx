import { useState } from "react"
import { ChevronRightIcon, MapPinIcon, PlusIcon, TractorIcon } from "lucide-react"
import { Link } from "react-router"

import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { FarmFormDrawer } from "@/components/farm-form"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatDate } from "@/lib/format"
import { useFarms } from "@/lib/queries"

export default function FarmsPage() {
  const farms = useFarms()
  const [creating, setCreating] = useState(false)

  const newButton = (
    <Button onClick={() => setCreating(true)}>
      <PlusIcon data-icon="inline-start" />
      Nova fazenda
    </Button>
  )

  return (
    <>
      <PageHeader title="Fazendas" description="De onde vem o leite que você acompanha." actions={farms.data?.length ? newButton : undefined} />

      {farms.isPending && <ListSkeleton />}
      {farms.isError && <ErrorAlert error={farms.error} title="Não foi possível carregar as fazendas" />}

      {farms.data?.length === 0 && (
        <EmptyState title="Nenhuma fazenda ainda" description="Cadastre a primeira para registrar tanques e analisar o leite." action={newButton} />
      )}

      {farms.data && farms.data.length > 0 && (
        <ul className="flex flex-col gap-3">
          {farms.data.map((farm) => (
            <li key={farm.id}>
              <Link to={`/fazendas/${farm.id}`} className="block rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
                <Card className="transition-colors hover:bg-accent/40 active:bg-accent/60">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <TractorIcon className="size-4 text-primary" />
                      <span className="truncate">{farm.name}</span>
                    </CardTitle>
                    <CardDescription className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <span className="flex items-center gap-1">
                        <MapPinIcon className="size-3.5" />
                        {farm.city} - {farm.state}
                      </span>
                      <span>Desde {formatDate(farm.created_at)}</span>
                    </CardDescription>
                    <CardAction>
                      <ChevronRightIcon className="size-5 text-muted-foreground" />
                    </CardAction>
                  </CardHeader>
                </Card>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <FarmFormDrawer open={creating} onOpenChange={setCreating} />
    </>
  )
}
