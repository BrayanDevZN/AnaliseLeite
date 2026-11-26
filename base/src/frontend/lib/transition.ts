import { useSyncExternalStore } from "react"

/** Qual transição aparece: entrando, saindo ou indo para o Google. */
export type TransitionKind = "in" | "out" | "google"

interface State {
  kind: TransitionKind | null
  /** Muda a cada pedido, para a animação recomeçar mesmo com o mesmo tipo. */
  id: number
}

const listeners = new Set<() => void>()
let state: State = { kind: null, id: 0 }

const GOOGLE_FLAG = "analise-leite:google-login"

function set(next: State) {
  state = next
  listeners.forEach((listener) => listener())
}

/** Mostra a transição em tela cheia (ela some sozinha). */
export function showTransition(kind: TransitionKind) {
  set({ kind, id: state.id + 1 })
}

/** Esconde a transição (ex.: o login falhou depois de ela aparecer). */
export function hideTransition() {
  if (state.kind !== null) set({ kind: null, id: state.id })
}

/** Marca que a pessoa saiu para entrar com o Google: na volta, a transição de "entrando" aparece. */
export function markGoogleLogin() {
  try {
    sessionStorage.setItem(GOOGLE_FLAG, "1")
  } catch {
    // sem sessionStorage (janela privada): só perde a transição na volta
  }
}

/** Lê e apaga a marca do login com o Google. */
export function consumeGoogleLogin(): boolean {
  try {
    const marked = sessionStorage.getItem(GOOGLE_FLAG) === "1"

    sessionStorage.removeItem(GOOGLE_FLAG)

    return marked
  } catch {
    return false
  }
}

export function useTransition(): State {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener)

      return () => listeners.delete(listener)
    },
    () => state,
  )
}
