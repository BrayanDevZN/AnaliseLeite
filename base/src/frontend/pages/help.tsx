import { BookOpenIcon, FlaskConicalIcon, MilkIcon, PlusIcon, ShieldCheckIcon, TractorIcon, UserRoundIcon, UsersIcon, type LucideIcon } from "lucide-react"
import { Link } from "react-router"

import { PageHeader } from "@/components/page-header"
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useAuth } from "@/lib/auth"
import { cn } from "@/lib/utils"

interface Step {
  icon: LucideIcon
  tone: string
  title: string
  text: string
  to: string
  action: string
}

/** Cada passo tem um tom da paleta, o mesmo do ícone dele no menu. */
const STEPS: Step[] = [
  {
    icon: TractorIcon,
    tone: "bg-chart-3/15 text-chart-3",
    title: "Cadastre uma fazenda",
    text: "Toda análise pertence a uma fazenda. Informe o nome, a cidade e o estado, e, se quiser, uma foto.",
    to: "/fazendas",
    action: "Ir para Fazendas",
  },
  {
    icon: MilkIcon,
    tone: "bg-primary/12 text-primary",
    title: "Registre os tanques",
    text: "Dentro da fazenda, cadastre o tanque onde o leite é guardado, com o tipo e a capacidade em litros.",
    to: "/fazendas",
    action: "Ver tanques",
  },
  {
    icon: FlaskConicalIcon,
    tone: "bg-chart-5/15 text-chart-5",
    title: "Faça uma análise",
    text: "Envie uma amostra preenchida na tela ou um arquivo com várias amostras (.csv ou .parquet). O modelo responde se o leite está aprovado ou reprovado.",
    to: "/analises/nova",
    action: "Nova análise",
  },
  {
    icon: ShieldCheckIcon,
    tone: "bg-butter/35 text-accent-foreground",
    title: "Acompanhe os resultados",
    text: "O Início mostra o resumo: quantas fazendas, tanques e análises você tem, a taxa de aprovação e a atividade dos últimos 7 dias.",
    to: "/",
    action: "Ver o Início",
  },
]

const GLOSSARY = [
  { term: "CCS", meaning: "Contagem de células somáticas. Mede a saúde do úbere: quanto maior, pior a qualidade do leite." },
  { term: "CPP", meaning: "Contagem padrão em placa. Mede a quantidade de bactérias: quanto maior, mais contaminado está o leite." },
  { term: "Temperatura", meaning: "A temperatura do leite no tanque, em °C. Leite quente estraga mais rápido." },
  { term: "Aprovado / Reprovado", meaning: "A resposta do modelo para a amostra, com base no que ele aprendeu de coletas anteriores." },
]

const FAQ = [
  {
    question: "O resultado é um laudo oficial?",
    answer: "Não. O modelo faz uma estimativa a partir dos dados da coleta, para ajudar a decidir onde olhar primeiro. O exame de laboratório continua valendo.",
  },
  {
    question: "Como envio várias amostras de uma vez?",
    answer: "Em Nova análise, escolha a opção de arquivo e envie um .csv ou .parquet. O resultado mostra quantas amostras foram aprovadas e reprovadas, e cada linha com a resposta dela.",
  },
  {
    question: "Posso escolher outro modelo?",
    answer: "Sim. Se houver mais de um modelo disponível, a tela de Nova análise deixa escolher qual usar. Sem escolha, vale o modelo padrão.",
  },
  {
    question: "Apaguei uma fazenda sem querer. E agora?",
    answer: "Apagar não tem volta. Por isso o sistema sempre pede uma confirmação antes. Cadastre a fazenda de novo e refaça as análises, se precisar.",
  },
  {
    question: "Como troco minha foto ou minha senha?",
    answer: "Na aba Perfil. Lá também fica o botão para sair da conta.",
  },
  {
    question: "Entrei com o Google. Preciso de senha?",
    answer: "Não. Quem entra com o Google não precisa de senha. Se quiser uma, é só definir no Perfil.",
  },
]

export default function HelpPage() {
  const { user } = useAuth()

  return (
    <>
      <PageHeader title="Como usar" description="Para que serve o Análise Leite e como tirar o melhor dele." />

      <Card className="depth">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpenIcon className="size-5 text-primary" />
            Para que serve
          </CardTitle>
          <CardDescription>Acompanhar a qualidade do leite, da fazenda ao laticínio.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 text-sm leading-relaxed">
          <p>
            Quando leite fora do padrão entra em um tanque compartilhado, ele se mistura com o de outras fazendas, segue no caminhão e chega ao silo. Sem dados, é difícil saber de onde veio o problema e quem
            deve arcar com ele.
          </p>
          <p>
            O <strong>Análise Leite</strong> reúne as fazendas, os tanques e as coletas em um só lugar, e usa um modelo de aprendizado de máquina para estimar, a partir da temperatura, da CCS e da CPP, se uma
            amostra seria <strong>aprovada</strong> ou <strong>reprovada</strong> no exame.
          </p>
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">Fazendas e tanques</Badge>
            <Badge variant="secondary">Análise por amostra ou arquivo</Badge>
            <Badge variant="secondary">Resumo e histórico</Badge>
          </div>
        </CardContent>
      </Card>

      <section className="flex flex-col gap-3" aria-labelledby="steps-title">
        <h2 id="steps-title" className="font-heading text-lg font-semibold">
          Passo a passo
        </h2>
        <ol className="flex flex-col gap-3">
          {STEPS.map((step, index) => (
            <li key={step.title}>
              <Card size="sm" className="depth">
                <CardHeader className="flex flex-row items-start gap-3">
                  <span className={cn("flex size-10 shrink-0 items-center justify-center rounded-full", step.tone)}>
                    <step.icon className="size-5" aria-hidden="true" />
                  </span>
                  <div className="flex min-w-0 flex-1 flex-col gap-1">
                    <CardTitle>
                      {index + 1}. {step.title}
                    </CardTitle>
                    <CardDescription>{step.text}</CardDescription>
                    <Link to={step.to} className={cn(buttonVariants({ variant: "outline", size: "sm" }), "mt-2 w-fit")}>
                      {index === 2 && <PlusIcon data-icon="inline-start" />}
                      {step.action}
                    </Link>
                  </div>
                </CardHeader>
              </Card>
            </li>
          ))}
        </ol>
      </section>

      <section className="flex flex-col gap-3" aria-labelledby="glossary-title">
        <h2 id="glossary-title" className="font-heading text-lg font-semibold">
          O que significa cada campo
        </h2>
        <Card size="sm" className="depth">
          <CardContent>
            <dl className="flex flex-col gap-3 text-sm">
              {GLOSSARY.map((item) => (
                <div key={item.term} className="flex flex-col gap-0.5">
                  <dt className="font-semibold text-primary">{item.term}</dt>
                  <dd className="text-muted-foreground">{item.meaning}</dd>
                </div>
              ))}
            </dl>
          </CardContent>
        </Card>
      </section>

      {user?.role === "admin" && (
        <Card size="sm" className="depth">
          <CardHeader className="flex flex-row items-start gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-chart-4/15 text-chart-4">
              <UsersIcon className="size-5" aria-hidden="true" />
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <CardTitle>Para administradores</CardTitle>
              <CardDescription>
                Na aba Usuários você vê todas as pessoas, cria contas, define quem é administrador e abre cada uma para ver as fazendas e os tanques dela. Apagar um usuário também apaga as fazendas, os
                tanques e as análises dele.
              </CardDescription>
              <Link to="/usuarios" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "mt-2 w-fit")}>
                Ir para Usuários
              </Link>
            </div>
          </CardHeader>
        </Card>
      )}

      <section className="flex flex-col gap-3" aria-labelledby="faq-title">
        <h2 id="faq-title" className="font-heading text-lg font-semibold">
          Perguntas frequentes
        </h2>
        <Card size="sm" className="depth">
          <CardContent>
            <Accordion>
              {FAQ.map((item) => (
                <AccordionItem key={item.question} value={item.question}>
                  <AccordionTrigger>{item.question}</AccordionTrigger>
                  <AccordionContent className="text-muted-foreground">{item.answer}</AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </CardContent>
        </Card>
      </section>

      <Link to="/perfil" className={cn(buttonVariants({ variant: "outline", size: "lg" }))}>
        <UserRoundIcon data-icon="inline-start" />
        Ir para o meu perfil
      </Link>
    </>
  )
}
