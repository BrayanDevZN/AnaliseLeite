import { useState } from "react"
import { CircleCheckIcon, CircleXIcon } from "lucide-react"

import { ApprovalBar } from "@/components/analysis-summary"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatNumber } from "@/lib/format"

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

const PAGE = 20

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

/** Resultado de um arquivo: quantas linhas passaram e uma lista (paginada) com a resposta de cada uma. */
export function RowsResult({ rows }: { rows: Record<string, unknown>[] }) {
  const [visible, setVisible] = useState(PAGE)

  const predict = (row: Record<string, unknown>) => String(row.predict ?? "")
  const approved = rows.filter((row) => predict(row).toLowerCase() === "aprovado").length
  const rejected = rows.filter((row) => predict(row).toLowerCase() === "reprovado").length

  return (
    <div className="flex flex-col gap-4">
      <ResultTotals approved={approved} rejected={rejected} />

      <ul className="flex flex-col gap-2">
        {rows.slice(0, visible).map((row, index) => (
          <li key={index}>
            <Card size="sm">
              <CardContent className="flex flex-col gap-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium">Amostra {index + 1}</span>
                  <StatusBadge value={predict(row)} />
                </div>
                <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                  {LABELS.filter(([key]) => row[key] != null).map(([key, label]) => (
                    <div key={key} className="flex min-w-0 flex-col">
                      <dt className="text-muted-foreground">{label}</dt>
                      <dd className="truncate font-medium">{show(row[key])}</dd>
                    </div>
                  ))}
                </dl>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>

      {visible < rows.length && (
        <Button variant="outline" size="lg" onClick={() => setVisible((count) => count + PAGE)}>
          Mostrar mais ({formatNumber(rows.length - visible)} restantes)
        </Button>
      )}
    </div>
  )
}
