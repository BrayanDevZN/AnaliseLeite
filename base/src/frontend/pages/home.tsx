import { FlaskConicalIcon, MilkIcon, PlusIcon, ShieldCheckIcon, TractorIcon } from "lucide-react"
import { Link } from "react-router"

import { AnalysisCard, ApprovalBar } from "@/components/analysis-summary"
import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { ProfilePhoto } from "@/components/profile-photo"
import { StatCard } from "@/components/stat-card"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { WeekChart } from "@/components/week-chart"
import { useAuth } from "@/lib/auth"
import { formatNumber } from "@/lib/format"
import { useMyAnalyses, useStats } from "@/lib/queries"
import { cn } from "@/lib/utils"

/** Bom dia, boa tarde ou boa noite, pela hora do aparelho. */
function greeting(now = new Date()): string {
  const hour = now.getHours()

  if (hour < 12) return "Bom dia"
  if (hour < 18) return "Boa tarde"

  return "Boa noite"
}

/** A página de início: boas-vindas com a foto, um resumo (dashboard) e as análises mais recentes. */
export default function HomePage() {
  const { user } = useAuth()
  const stats = useStats()
  const recent = useMyAnalyses()

  if (!user) return null

  const firstName = user.name.trim().split(/\s+/)[0]
  const data = stats.data
  const samples = (data?.approved ?? 0) + (data?.rejected ?? 0)
  const rate = samples === 0 ? null : Math.round(((data?.approved ?? 0) / samples) * 100)
  const latest = recent.data?.pages[0]?.items.slice(0, 5) ?? []
  const isNew = data !== undefined && data.farms === 0 && data.analyses === 0

  return (
    <>
      <Card>
        <CardContent className="flex items-center gap-4">
          <ProfilePhoto user={user} className="size-20" />
          <div className="flex min-w-0 flex-col gap-0.5">
            <p className="text-sm text-muted-foreground">{greeting()},</p>
            <h1 className="truncate font-heading text-2xl font-semibold tracking-tight">{firstName}!</h1>
            <p className="text-sm text-muted-foreground">
              {isNew ? "Boas-vindas ao Análise Leite. Vamos começar?" : "Boas-vindas de volta ao Análise Leite."}
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-2 gap-3">
        <Link to="/analises/nova" className={cn(buttonVariants({ size: "lg" }))}>
          <PlusIcon data-icon="inline-start" />
          Nova análise
        </Link>
        <Link to="/fazendas" className={cn(buttonVariants({ variant: "outline", size: "lg" }))}>
          <TractorIcon data-icon="inline-start" />
          Fazendas
        </Link>
      </div>

      {stats.isError && <ErrorAlert error={stats.error} title="Não foi possível carregar o resumo" />}

      {stats.isPending && (
        <div className="grid grid-cols-2 gap-3" aria-busy="true" aria-label="Carregando o resumo">
          {Array.from({ length: 4 }, (_, index) => (
            <Skeleton key={index} className="h-24 rounded-xl" />
          ))}
        </div>
      )}

      {data && isNew && (
        <EmptyState
          title="Comece pela sua primeira fazenda"
          description="Cadastre a fazenda e os tanques, e depois envie uma amostra ou um arquivo para o modelo analisar."
          action={
            <Link to="/fazendas" className={cn(buttonVariants({ size: "lg" }))}>
              Cadastrar fazenda
            </Link>
          }
        />
      )}

      {data && !isNew && (
        <>
          <div className="grid grid-cols-2 gap-3">
            <StatCard icon={TractorIcon} label="Fazendas" value={formatNumber(data.farms)} />
            <StatCard icon={MilkIcon} label="Tanques" value={formatNumber(data.tanks)} />
            <StatCard icon={FlaskConicalIcon} label="Análises" value={formatNumber(data.analyses)} />
            <StatCard icon={ShieldCheckIcon} label="Aprovação" value={rate === null ? "—" : `${rate}%`} hint={rate === null ? "sem resultados ainda" : "das amostras"} />
          </div>

          {samples > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Resultado das amostras</CardTitle>
                <CardDescription>O que o modelo aprovou e reprovou, somando tudo o que você analisou.</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <ApprovalBar approved={data.approved} rejected={data.rejected} />
                <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-sm font-medium">
                  <span className="text-success">{formatNumber(data.approved)} aprovadas</span>
                  <span className="text-destructive">{formatNumber(data.rejected)} reprovadas</span>
                </div>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Últimos 7 dias</CardTitle>
              <CardDescription>Quantas análises você fez por dia.</CardDescription>
            </CardHeader>
            <CardContent>
              <WeekChart days={data.days} />
            </CardContent>
          </Card>
        </>
      )}

      {latest.length > 0 && (
        <section className="flex flex-col gap-3" aria-labelledby="recent-title">
          <div className="flex items-center justify-between gap-3">
            <h2 id="recent-title" className="font-heading text-lg font-semibold">
              Análises recentes
            </h2>
            <Link to="/analises" className="text-sm font-medium text-primary underline-offset-4 hover:underline">
              Ver todas
            </Link>
          </div>
          <ul className="flex flex-col gap-3">
            {latest.map((analysis) => (
              <li key={analysis.id}>
                <AnalysisCard analysis={analysis} />
              </li>
            ))}
          </ul>
        </section>
      )}
    </>
  )
}
