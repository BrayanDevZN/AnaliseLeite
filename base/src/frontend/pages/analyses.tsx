import { PlusIcon } from "lucide-react"
import { Link } from "react-router"

import { AnalysisCard } from "@/components/analysis-summary"
import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { Button, buttonVariants } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { formatNumber } from "@/lib/format"
import { useFarms, useMyAnalyses } from "@/lib/queries"
import { cn } from "@/lib/utils"

/** Todas as análises que a pessoa fez, de todas as fazendas, com o resultado de cada uma. */
export default function AnalysesPage() {
  const analyses = useMyAnalyses()
  const farms = useFarms()

  const items = analyses.data?.pages.flatMap((page) => page.items) ?? []
  const total = analyses.data?.pages[0]?.total ?? 0

  const newButton = (
    <Link to="/analises/nova" className={cn(buttonVariants())}>
      <PlusIcon data-icon="inline-start" />
      Nova análise
    </Link>
  )

  return (
    <>
      <PageHeader
        title="Análises"
        description={total > 0 ? `${formatNumber(total)} ${total === 1 ? "análise feita" : "análises feitas"}, com o resultado de cada uma.` : "Tudo o que você analisou fica aqui."}
        actions={items.length > 0 ? newButton : undefined}
      />

      {analyses.isPending && <ListSkeleton />}
      {analyses.isError && <ErrorAlert error={analyses.error} title="Não foi possível carregar as análises" />}

      {analyses.data && items.length === 0 && (
        <EmptyState
          title="Nenhuma análise ainda"
          description={
            farms.data?.length === 0
              ? "Cadastre uma fazenda primeiro, depois envie uma amostra ou um arquivo para o modelo."
              : "Envie uma amostra ou um arquivo para o modelo dizer se o leite está aprovado ou reprovado."
          }
          action={
            farms.data?.length === 0 ? (
              <Link to="/fazendas" className={cn(buttonVariants({ size: "lg" }))}>
                Cadastrar fazenda
              </Link>
            ) : (
              <Link to="/analises/nova" className={cn(buttonVariants({ size: "lg" }))}>
                Nova análise
              </Link>
            )
          }
        />
      )}

      {items.length > 0 && (
        <ul className="flex flex-col gap-3">
          {items.map((analysis) => (
            <li key={analysis.id}>
              <AnalysisCard analysis={analysis} />
            </li>
          ))}
        </ul>
      )}

      {analyses.hasNextPage && (
        <Button variant="outline" size="lg" disabled={analyses.isFetchingNextPage} onClick={() => analyses.fetchNextPage()}>
          {analyses.isFetchingNextPage && <Spinner data-icon="inline-start" />}
          Mostrar mais ({formatNumber(total - items.length)} restantes)
        </Button>
      )}
    </>
  )
}
