import { ChevronRightIcon, FileSpreadsheetIcon, FlaskConicalIcon } from "lucide-react"
import { Link } from "react-router"

import { StatusBadge } from "@/components/status-badge"
import { Badge } from "@/components/ui/badge"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatDateTime, formatNumber } from "@/lib/format"
import type { Analysis } from "@/lib/types"

/** Barra de aprovadas (verde) e reprovadas (vermelho). */
export function ApprovalBar({ approved, rejected }: { approved: number; rejected: number }) {
  const total = approved + rejected
  const percent = total === 0 ? 0 : (approved / total) * 100

  return (
    <div
      role="img"
      aria-label={`${Math.round(percent)}% aprovadas`}
      className="flex h-2 w-full overflow-hidden rounded-full bg-muted"
    >
      <div className="bg-success" style={{ width: `${percent}%` }} />
      <div className="bg-destructive" style={{ width: `${100 - percent}%` }} />
    </div>
  )
}

/**
 * O resultado de uma análise em poucas linhas: um selo para uma amostra, uma barra com as contagens para um arquivo,
 * e um aviso nas análises antigas, de antes de o resultado ser guardado.
 */
export function ResultSummary({ analysis }: { analysis: Pick<Analysis, "kind" | "approved" | "rejected"> }) {
  const total = analysis.approved + analysis.rejected

  if (total === 0) {
    return <Badge variant="outline">Resultado não guardado</Badge>
  }

  if (analysis.kind === "sample") {
    return <StatusBadge value={analysis.approved > 0 ? "aprovado" : "reprovado"} />
  }

  return (
    <div className="flex w-full flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs font-medium">
        <span className="text-success">{formatNumber(analysis.approved)} aprovadas</span>
        <span className="text-destructive">{formatNumber(analysis.rejected)} reprovadas</span>
      </div>
      <ApprovalBar approved={analysis.approved} rejected={analysis.rejected} />
    </div>
  )
}

/** Um cartão de análise para as listas: o que foi, de qual fazenda, quando e o resultado. Leva para o detalhe. */
export function AnalysisCard({ analysis }: { analysis: Analysis }) {
  const Icon = analysis.kind === "file" ? FileSpreadsheetIcon : FlaskConicalIcon

  return (
    <Link to={`/analises/${analysis.id}`} className="block rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      <Card size="sm" className="transition-colors hover:bg-accent/40 active:bg-accent/60">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Icon className="size-4 text-primary" />
            {analysis.kind === "file" ? "Arquivo" : "Amostra"}
            <span className="font-normal text-muted-foreground">#{analysis.id}</span>
          </CardTitle>
          <CardDescription className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
            {analysis.farm_name && <span className="font-medium text-foreground">{analysis.farm_name}</span>}
            <span>{formatDateTime(analysis.created_at)}</span>
            {analysis.model && <span>{analysis.model}</span>}
          </CardDescription>
          <CardAction>
            <ChevronRightIcon className="size-5 text-muted-foreground" />
          </CardAction>
        </CardHeader>
        <CardContent>
          <ResultSummary analysis={analysis} />
        </CardContent>
      </Card>
    </Link>
  )
}
