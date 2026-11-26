import { FileSpreadsheetIcon, Trash2Icon } from "lucide-react"
import { Link, Navigate, useNavigate, useParams } from "react-router"

import { ConfirmDelete } from "@/components/confirm-delete"
import { ErrorAlert } from "@/components/error-alert"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ApiError } from "@/lib/api"
import { formatBytes, formatDateTime, formatNumber } from "@/lib/format"
import { notify } from "@/lib/notify"
import { useAnalysis, useContents, useDeleteAnalysis, useFarm, useTanks } from "@/lib/queries"
import type { Content } from "@/lib/types"

const LABELS: [keyof NonNullable<Content["content"]>, string][] = [
  ["data", "Data"],
  ["temperature", "Temperatura (°C)"],
  ["ccs", "CCS"],
  ["cpp", "CPP"],
  ["tanque", "Código do tanque"],
  ["cidade", "Cidade"],
  ["state", "UF"],
]

export default function AnalysisDetailPage() {
  const id = Number(useParams().id)
  const navigate = useNavigate()

  const analysis = useAnalysis(id)
  const contents = useContents(id)
  const farmId = analysis.data?.farm_id ?? 0
  const farm = useFarm(farmId)
  const tanks = useTanks(farmId)
  const remove = useDeleteAnalysis(farmId)

  if (!Number.isInteger(id) || id <= 0 || (analysis.error instanceof ApiError && analysis.error.status === 404)) {
    return <Navigate to="/fazendas" replace />
  }

  if (analysis.isError) return <ErrorAlert error={analysis.error} title="Não foi possível carregar a análise" />

  if (analysis.isPending) return <ListSkeleton count={3} />

  const tank = tanks.data?.find((item) => item.id === analysis.data.tank_id)

  async function apagar() {
    try {
      await remove.mutateAsync(id)
      notify.success("Análise apagada")
      navigate(`/fazendas/${farmId}`, { replace: true })
    } catch (error) {
      notify.error("Não foi possível apagar", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <>
      <PageHeader
        title={`Análise #${id}`}
        description={formatDateTime(analysis.data.created_at)}
        back={`/fazendas/${farmId}`}
        actions={
          <ConfirmDelete
            trigger={
              <Button variant="outline" size="icon" aria-label="Apagar análise">
                <Trash2Icon />
              </Button>
            }
            title="Apagar esta análise?"
            description="A análise e os dados enviados nela serão apagados."
            onConfirm={apagar}
          />
        }
      />

      <Card size="sm">
        <CardContent className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-muted-foreground">Fazenda</span>
          <Link to={`/fazendas/${farmId}`} className="font-medium text-primary underline-offset-4 hover:underline">
            {farm.data?.name ?? `#${farmId}`}
          </Link>
          {tank && (
            <>
              <span className="text-muted-foreground">Tanque</span>
              <Badge variant="outline" className="capitalize">
                {tank.type}
              </Badge>
            </>
          )}
        </CardContent>
      </Card>

      <section className="flex flex-col gap-3" aria-labelledby="contents-title">
        <h2 id="contents-title" className="font-heading text-lg font-semibold">
          Dados enviados
        </h2>

        {contents.isPending && <ListSkeleton count={1} />}
        {contents.isError && <ErrorAlert error={contents.error} />}

        {contents.data?.map((content) => (
          <Card key={content.id}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileSpreadsheetIcon className="size-4 text-primary" />
                {content.kind === "file" ? "Arquivo" : "Amostra"}
              </CardTitle>
              <CardDescription>{formatDateTime(content.created_at)}</CardDescription>
            </CardHeader>
            <CardContent>
              {content.kind === "file" ? (
                <p className="text-sm">Arquivo de {formatBytes(content.file_size ?? 0)}.</p>
              ) : (
                <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                  {LABELS.filter(([key]) => content.content?.[key] != null).map(([key, label]) => (
                    <div key={key} className="flex min-w-0 flex-col">
                      <dt className="text-xs text-muted-foreground">{label}</dt>
                      <dd className="truncate font-medium">{typeof content.content?.[key] === "number" ? formatNumber(content.content[key] as number) : String(content.content?.[key])}</dd>
                    </div>
                  ))}
                </dl>
              )}
            </CardContent>
          </Card>
        ))}
      </section>
    </>
  )
}
