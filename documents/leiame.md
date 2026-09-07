# O sistema que a cooperativa não tem

**Projeto e Engenharia de Software · A3 · 40 pontos**

Este arquivo vale para os cinco grupos. O problema do seu grupo está em
`enunciados/`, e os seus arquivos de dados estão em `base/grupo-N/`.

---

## O cenário

Uma cooperativa de leite. Dezenas de fazendas ordenham e despejam o leite num
tanque de resfriamento — algumas têm tanque próprio, outras dividem um tanque
comunitário com os vizinhos. Um caminhão recolhe vários tanques ao longo de uma
rota e descarrega num silo do laticínio, onde outros caminhões também
descarregam.

A cooperativa funciona. O que ela não tem é **sistema**. Tudo o que acontece
nessa cadeia está anotado em caderno, em planilha solta, em mensagem de celular
ou na cabeça de alguém. Quando dá problema, ninguém consegue reconstruir o que
aconteceu.

**O trabalho de vocês é especificar o sistema que faltava.** Não é programar —
é entender o problema, escrever o que o sistema precisa fazer, desenhar como
ele seria e mostrar o que a gestão passaria a enxergar.

## Vocês não vão escrever código

Zero linhas. Isso não é um trabalho reduzido: é o trabalho que vem **antes** do
código, e é onde os projetos de software são ganhos ou perdidos. Quem programa
recebe o desenho pronto e executa. Desenho errado custa caro depois.

O que vocês produzem é o que uma pessoa de análise entrega numa empresa: um
documento com o problema entendido, os requisitos escritos, os diagramas da
solução e um painel com os números.

---

## As quatro fases

| Fase | O que vocês fazem | O que fica escrito |
|---|---|---|
| **1 · Entender o problema** | Ouvem o cliente e investigam a causa | Ishikawa e SWOT |
| **2 · Levantar requisitos** | Perguntam ao cliente e escrevem o que o sistema precisa fazer | Lista de requisitos |
| **3 · Modelar a solução** | Desenham o sistema em UML | Casos de uso e classes |
| **4 · Medir** | Montam o painel sobre os dados | Relatório do Power BI |

**O cliente é o professor.** Nas aulas de levantamento de requisitos ele entra
no papel do gestor da cooperativa e responde ao que vocês perguntarem. Ele não
vai adivinhar o que vocês precisam saber — pergunta que não é feita vira
requisito que não existe.

## O entregável: um relatório em PDCA

**Um documento só**, publicado no GitHub, organizado em quatro partes:

| | O que vai dentro |
|---|---|
| **P** — Plan | O problema, o Ishikawa, a SWOT e a lista de requisitos |
| **D** — Do | Os diagramas de casos de uso e de classes |
| **C** — Check | O painel e o que cada indicador mostra |
| **A** — Act | O que ficou faltando e o que fariam diferente |

O **A** não é enfeite. É onde vocês escrevem o que não deu tempo, o que ficou
frágil e o que mudariam. Relatório que só conta acerto não convence ninguém que
já trabalhou.

## O que fica no repositório

1. **`relatorio.md`** — o relatório em PDCA, com os diagramas embutidos
2. **`relatorio.pbix`** — o arquivo do Power BI
3. **`relatorio.pdf`** — o painel exportado, para quem não tem o Power BI
4. **`diagramas/`** — os PNG dos diagramas de casos de uso e de classes
5. **`base/`** — os arquivos de dados que vocês receberam
6. **`mascote.png`** — o mascote do grupo

**O mascote não é enfeite.** Todo time de software que se leva a sério tem um
bicho, um nome e uma piada interna. Escolham um que tenha a ver com o problema
de vocês, deem um nome, escrevam uma linha explicando, e coloquem no alto do
README — antes do texto sério.

---

## Os arquivos de dados

Cada grupo recebe **dois ou três** arquivos CSV, e eles precisam ser
relacionados dentro do Power BI:

- uma tabela de **movimento** — o que aconteceu, muitas linhas;
- uma ou duas tabelas de **cadastro** — quem é quem, poucas linhas.

Elas se ligam por uma coluna de código, e a ligação não é burocracia:
**informação que você precisa para um indicador está no outro arquivo.** Sem
relacionar as tabelas na visão de Modelo do Power BI, o número simplesmente não
aparece.

Os arquivos estão limpos: nenhuma célula vazia, nenhum código solto, nenhuma
linha sobrando de um lado. O trabalho aqui é montar o painel, não consertar
dado.

> **De onde vêm esses dados?** Eles são fictícios, e representam o arquivo que
> *o sistema que vocês estão especificando* teria produzido, se estivesse
> rodando há seis meses. É por isso que o painel fecha o trabalho: ele mostra o
> que a cooperativa passaria a enxergar depois que o sistema existisse.

Separador `;`, decimal com vírgula, datas em AAAA-MM-DD. Abre direto no Power
BI e no Excel, sem configurar nada.

---

## Glossário — o mínimo para entender o problema

| Termo | O que é |
|---|---|
| **Tanque de expansão** | O tanque que resfria o leite a 4 °C e o guarda até o caminhão chegar |
| **Tanque comunitário** | Um tanque dividido por vários vizinhos: o leite se mistura antes da coleta |
| **Carga** | Tudo que está dentro do caminhão. O compartimento é um só: o que entra se mistura |
| **Silo** | O tanque grande do laticínio, onde vários caminhões descarregam |
| **Rota** | A sequência de fazendas que um caminhão visita no dia |
| **Romaneio** | O papel que registra o que foi coletado em cada ponto da rota |
| **CCS** | Contagem de células somáticas. Quanto **menor**, melhor a qualidade |
| **CPP** | Contagem de bactérias. Quanto **menor**, melhor a higiene |
| **BPA** | Programa de boas práticas: um checklist que define quanto o produtor recebe de bônus por litro |
| **Carência** | Período em que o leite da vaca tratada com antibiótico não pode ser vendido |

---

## As entregas parciais

O trabalho corre em **sprints**. Cada sprint é uma data em que o professor abre
o repositório do grupo e confere o que está publicado.

1. A sprint fecha — vocês publicam o que existe;
2. o professor lê e compara com o que precisa existir;
3. ele devolve **o buraco**, e não a resposta: *«falta quem registra a saída do
   caminhão»*, não *«falta a classe Motorista»*;
4. vocês corrigem antes da sprint seguinte.

> **Se não está no repositório na noite da sprint, não existe.** Não há entrega
> atrasada e não há trabalho levado para casa: tudo é produzido e publicado na
> aula.

**Cada aluno publica pela própria conta.** O histórico do repositório mostra
quem fez o quê, e é dele que sai a nota individual — quem não publicou nada não
tem sobre o que ser arguido na apresentação.

## O que o professor confere, e o que ele não confere

| Ele confere | Ele **não** confere |
|---|---|
| Se o relatório está publicado | Se o desenho ficou igual ao dele |
| Se o desenho bate com os requisitos escritos | Quantas caixas o diagrama tem |
| Se o painel usa os dois arquivos relacionados | Se o mascote ficou bonito (MENTIRA, CONFIRO SIM!!!!) |
| Se cada integrante publicou alguma coisa | Quem escreveu mais texto |

**Não existe um desenho certo.** Dois analistas competentes modelam o mesmo
problema de formas diferentes, e as duas prestam. O que se cobra é
**coerência**: cada requisito que vocês escreveram precisa aparecer em algum
lugar do diagrama, e cada classe do diagrama precisa servir a algum requisito.

## Sobre usar IA

Não é proibido, e é assim no mercado. Mas há duas coisas que a ferramenta não
faz por vocês:

- **Ela não conversa com o cliente.** Quem descobre o que a cooperativa precisa
  é quem pergunta na aula. Requisito inventado por ferramenta não sobrevive à
  primeira pergunta do professor na apresentação.
- **Ela não defende o modelo.** Na apresentação, cada um responde sobre o que
  publicou. Desenho que ninguém do grupo sabe explicar vale zero, mesmo estando
  correto.

---