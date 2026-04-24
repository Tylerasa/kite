import { Deposit } from "./deposit"
import { Failed } from "./failed"
import { Payout } from "./payout"
import { Reversed } from "./reversed"
import { US } from "./us"
import { UK } from "./uk"
import { EUR } from "./eur"
import { Nigeria } from "./nigeria"
import { Kenya } from "./kenya"
import { Home } from "./home"
import { Send } from "./send"
import { Transactions } from "./transactions"
import { Transfer } from "./transfer"

export const Icons = {
  deposit: Deposit,
  failed: Failed,
  reversed: Reversed,
  payout: Payout,
  usd: US,
  gbp: UK,
  eur: EUR,
  ngn: Nigeria,
  kes: Kenya,
  home: Home,
  send: Send,
  transactions: Transactions,
  transfer: Transfer,
}
