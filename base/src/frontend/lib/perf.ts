/**
 * Detecta aparelho fraco (poucos núcleos, pouca memória ou economia de dados) e marca o <html> com `data-perf="low"`.
 * O CSS usa essa marca para aliviar as animações do fundo sem tirá-las.
 */
export function applyPerfMode() {
  const nav = navigator as Navigator & { deviceMemory?: number; connection?: { saveData?: boolean } }

  const low =
    (nav.hardwareConcurrency !== undefined && nav.hardwareConcurrency <= 4) ||
    (nav.deviceMemory !== undefined && nav.deviceMemory <= 4) ||
    nav.connection?.saveData === true

  if (low) document.documentElement.dataset.perf = "low"
}
