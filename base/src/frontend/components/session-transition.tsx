import { useEffect, useState } from "react"

import mascot from "@/assets/mascote.webp"
import { MilkOrbit } from "@/components/milk-orbit"
import { useAuth } from "@/lib/auth"
import { consumeGoogleLogin, hideTransition, showTransition, useTransition, type TransitionKind } from "@/lib/transition"

const TEXT: Record<TransitionKind, { title: string; subtitle: string }> = {
  in: { title: "Entrando...", subtitle: "Preparando suas fazendas e análises" },
  out: { title: "Saindo...", subtitle: "Até a próxima!" },
  google: { title: "Conectando ao Google...", subtitle: "Só um instante" },
}

/** Quanto a transição fica na tela (o login com o Google fica até a página sair; o limite é só uma garantia). */
const DURATION: Record<TransitionKind, number> = { in: 1500, out: 1200, google: 10_000 }
const FADE_OUT = 350

/**
 * Transição em tela cheia ao entrar ou sair: o mascote gira em 3D no meio das órbitas de leite, sobre um véu de cor.
 * Aparece no login com senha, no cadastro, no login com o Google (na ida e na volta) e ao sair.
 */
export function SessionTransition() {
  const { kind, id } = useTransition()
  const { user, loading } = useAuth()

  // volta do Google: a página recarregou e a pessoa já está logada
  useEffect(() => {
    if (loading) return

    if (consumeGoogleLogin() && user) showTransition("in")
  }, [loading, user])

  // voltar pelo botão do navegador a partir do Google traz a página como estava: tira o "Conectando..."
  useEffect(() => {
    const onShow = (event: PageTransitionEvent) => {
      if (event.persisted) hideTransition()
    }

    window.addEventListener("pageshow", onShow)

    return () => window.removeEventListener("pageshow", onShow)
  }, [])

  // key={id}: cada pedido monta uma tela nova, e a animação recomeça do zero
  return kind ? <Overlay key={id} kind={kind} /> : null
}

function Overlay({ kind }: { kind: TransitionKind }) {
  const [closing, setClosing] = useState(false)
  const text = TEXT[kind]

  // some sozinha: primeiro esmaece, depois sai
  useEffect(() => {
    const fade = window.setTimeout(() => setClosing(true), DURATION[kind])
    const done = window.setTimeout(hideTransition, DURATION[kind] + FADE_OUT)

    return () => {
      window.clearTimeout(fade)
      window.clearTimeout(done)
    }
  }, [kind])

  return (
    <div
      role="status"
      aria-live="polite"
      data-closing={closing}
      className="fixed inset-0 z-[100] flex flex-col items-center justify-center gap-6 bg-background/80 backdrop-blur-xl data-[closing=false]:animate-in data-[closing=false]:fade-in data-[closing=true]:animate-out data-[closing=true]:fade-out data-[closing=true]:fill-mode-forwards duration-300"
      style={{ backgroundImage: "linear-gradient(135deg, color-mix(in oklch, var(--bg-a) 85%, transparent), color-mix(in oklch, var(--bg-b) 85%, transparent) 50%, color-mix(in oklch, var(--bg-c) 85%, transparent))" }}
    >
      <div className="relative flex size-48 items-center justify-center [perspective:800px]">
        <MilkOrbit className="absolute inset-4" />
        <img
          src={mascot}
          alt=""
          width={640}
          height={640}
          className="transition-spin relative size-28 rounded-full ring-4 ring-primary/30 shadow-xl shadow-primary/30"
        />
      </div>
      <div className="flex flex-col items-center gap-1 text-center">
        <p className="font-heading text-xl font-semibold tracking-tight">{text.title}</p>
        <p className="text-sm text-muted-foreground">{text.subtitle}</p>
      </div>
      <div className="h-1 w-40 overflow-hidden rounded-full bg-primary/15" aria-hidden="true">
        <div className="transition-bar h-full rounded-full bg-gradient-to-r from-primary via-butter to-chart-3" />
      </div>
    </div>
  )
}
