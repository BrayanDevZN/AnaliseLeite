import { useState } from "react"
import { CircleCheckIcon, CircleXIcon } from "lucide-react"

import { ApprovalBar } from "@/components/analysis-summary"
import { ResultFilter, type ResultFilterValue } from "@/components/result-filter"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"

/** Resultado de uma amostra: um selo grande e fácil de ler no celular. */
export function SampleResult({ result }: { result: string }) {
  const approved = result.toLowerCase() === "aprovado"
  const Icon = approved ? CircleCheckIcon : CircleXIcon

  return (
    <Card>
      <CardHeader className="items-center justify-items-center text-center">
        <Icon className={approved ? "size-14 text-success" : "size-14 text-destructive"} aria-hidden="true" />
        <CardTitle className="font-heading text-2xl">{approved ? "Leite aprovado" : "Leite reprovado"}</CardTitle>
        <CardDescription>
          {approved ? "O modelo não encontrou sinais de problema nesta amostra." : "O modelo encontrou sinais de problema nesta amostra. Vale uma conferência no laboratório."}
        </CardDescription>
      </CardHeader>
    </Card>
  )
}

const LABELS: [string, string][] = [
  ["data", "Data"],
  ["temperatura", "Temperatura"],
  ["ccs", "CCS"],
  ["cpp", "CPP"],
  ["tanque", "Tanque"],
  ["cidade", "Cidade"],
  ["estado", "UF"],
  ["resultado_exame", "Exame real"],
]

const PAGE = 25

function show(value: unknown): string {
  if (typeof value === "number") return formatNumber(value)

  return String(value).replace(/T00:00:00.*$/, "")
}

/** O resumo de um arquivo: quantas amostras o modelo aprovou e reprovou, e a barra. */
export function ResultTotals({ approved, rejected }: { approved: number; rejected: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-heading text-lg">{formatNumber(approved + rejected)} amostras analisadas</CardTitle>
        <CardDescription>Resumo da resposta do modelo para o arquivo enviado.</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1 rounded-lg bg-success/10 p-3">
            <span className="text-2xl font-semibold text-success">{formatNumber(approved)}</span>
            <span className="text-xs text-muted-foreground">aprovadas</span>
          </div>
          <div className="flex flex-col gap-1 rounded-lg bg-destructive/10 p-3">
            <span className="text-2xl font-semibold text-destructive">{formatNumber(rejected)}</span>
            <span className="text-xs text-muted-foreground">reprovadas</span>
          </div>
        </div>
        <ApprovalBar approved={approved} rejected={rejected} />
      </CardContent>
    </Card>
  )
}

/** Uma linha do arquivo, compacta: o resultado em destaque e os números principais numa linha só. */
function RowItem({ row, number }: { row: Record<string, unknown>; number: number }) {
  const approved = String(row.predict ?? "").toLowerCase() === "aprovado"
  const Icon = approved ? CircleCheckIcon : CircleXIcon
  const details = LABELS.filter(([key]) => row[key] != null && key !== "data")

  return (
    <li className="flex items-start gap-3 px-3 py-2.5">
      <Icon className={cn("mt-0.5 size-5 shrink-0", approved ? "text-success" : "text-destructive")} aria-hidden="true" />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">
            Amostra {number}
            {row.data != null && <span className="font-normal text-muted-foreground"> · {show(row.data)}</span>}
          </span>
          <StatusBadge value={String(row.predict ?? "")} />
        </div>
        <p className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
          {details.map(([key, label]) => (
            <span key={key}>
              {label} <strong className="font-medium text-foreground">{show(row[key])}</strong>
            </span>
          ))}
        </p>
      </div>
    </li>
  )
}

/**
 * Resultado de um arquivo: o resumo, um filtro (todas, aprovadas, reprovadas) e as amostras em linhas compactas
 * dentro de uma caixa com rolagem própria, para ler tudo sem descer a página inteira.
 */
export function RowsResult({ rows }: { rows: Record<string, unknown>[] }) {
  const [filter, setFilter] = useState<ResultFilterValue>("all")
  const [visible, setVisible] = useState(PAGE)

  const isApproved = (row: Record<string, unknown>) => String(row.predict ?? "").toLowerCase() === "aprovado"
  const isRejected = (row: Record<string, unknown>) => String(row.predict ?? "").toLowerCase() === "reprovado"

  // o número da amostra é o da posição no arquivo, mesmo com o filtro ligado
  const numbered = rows.map((row, index) => ({ row, number: index + 1 }))
  const approved = numbered.filter((item) => isApproved(item.row)).length
  const rejected = numbered.filter((item) => isRejected(item.row)).length
  const shown = numbered.filter((item) => filter === "all" || (filter === "approved" ? isApproved(item.row) : isRejected(item.row)))

  function change(next: ResultFilterValue) {
    setFilter(next)
    setVisible(PAGE)
  }

  return (
    <div className="flex flex-col gap-4">
      <ResultTotals approved={approved} rejected={rejected} />

      <section className="flex flex-col gap-3" aria-labelledby="rows-title">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h2 id="rows-title" className="font-heading text-lg font-semibold">
            Amostras do arquivo
          </h2>
          <ResultFilter value={filter} onChange={change} counts={{ all: rows.length, approved, rejected }} />
        </div>

        {shown.length === 0 ? (
          <Card size="sm">
            <CardContent className="text-sm text-muted-foreground">Nenhuma amostra {filter === "approved" ? "aprovada" : "reprovada"} neste arquivo.</CardContent>
          </Card>
        ) : (
          <Card size="sm" className="depth gap-0 overflow-hidden py-0">
            <ul className="max-h-[26rem] divide-y overflow-y-auto overscroll-contain">
              {shown.slice(0, visible).map((item) => (
                <RowItem key={item.number} row={item.row} number={item.number} />
              ))}
            </ul>
            <div className="flex items-center justify-between gap-2 border-t bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
              <span>
                Mostrando {formatNumber(Math.min(visible, shown.length))} de {formatNumber(shown.length)}
              </span>
              {visible < shown.length && (
                <Button variant="ghost" size="sm" onClick={() => setVisible((count) => count + PAGE)}>
                  Mostrar mais
                </Button>
              )}
            </div>
          </Card>
        )}
      </section>
    </div>
  )
}
