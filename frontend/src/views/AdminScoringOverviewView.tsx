import { useEffect, useMemo, useState } from 'react'
import { DeliverPolicy } from 'nats.ws'
import { getChecks } from '../services/api'
import { useNatsStore } from '../services/nats'
import ConnectionStatus from '../components/ConnectionStatus'
import './AdminScoringOverviewView.css'

const CELL_SEP = '\x1f'

const parseResultSubject = (subject: string): { team: string; check: string } | null => {
  const parts = subject.split('.')
  if (parts.length < 3 || parts[0] !== 'results') return null
  const rawCheck = parts.slice(2).join('.')
  const check = rawCheck.replace(/\.(0|1)$/, '')
  return { team: parts[1] ?? '', check }
}

const cellKey = (team: string, check: string) => `${team}${CELL_SEP}${check}`

export default function AdminScoringOverviewView() {
  const [columnChecks, setColumnChecks] = useState<string[]>([])
  const [checksLoadError, setChecksLoadError] = useState<string | null>(null)

  const status = useNatsStore(s => s.status)
  const messages = useNatsStore(s => s.messages)
  const error = useNatsStore(s => s.error)
  const connect = useNatsStore(s => s.connect)
  const subscribeToStream = useNatsStore(s => s.subscribeToStream)
  const clearMessages = useNatsStore(s => s.clearMessages)

  useEffect(() => {
    const run = async () => {
      if (status === 'disconnected') {
        await connect()
      }
    }
    void run()
  }, [])

  useEffect(() => {
    if (status === 'connected') {
      clearMessages()
      void subscribeToStream('results.>', { deliverPolicy: DeliverPolicy.LastPerSubject })
    }
  }, [status, subscribeToStream, clearMessages])

  useEffect(() => {
    const load = async () => {
      try {
        const names = await getChecks()
        setColumnChecks(names)
        setChecksLoadError(null)
      } catch (e) {
        setColumnChecks([])
        setChecksLoadError(e instanceof Error ? e.message : 'Failed to load checks')
      }
    }
    void load()
  }, [])

  const checkSet = useMemo(() => new Set(columnChecks), [columnChecks])

  const { teamsSorted, latestPassedByCell } = useMemo(() => {
    const teams = new Set<string>()
    /** Last message in buffer wins per team+check (delivery order ≈ time). */
    const latestPassed = new Map<string, boolean>()

    for (const msg of messages) {
      const parsed = parseResultSubject(msg.subject)
      if (!parsed) continue

      let passed = false
      try {
        const data = JSON.parse(msg.payload) as { passed?: boolean }
        passed = data.passed === true
      } catch {
        continue
      }

      teams.add(parsed.team)
      if (!checkSet.has(parsed.check)) continue

      const key = cellKey(parsed.team, parsed.check)
      latestPassed.set(key, passed)
    }

    const teamsSorted = Array.from(teams).sort((a, b) =>
      a.localeCompare(b, undefined, { numeric: true })
    )

    return { teamsSorted, latestPassedByCell: latestPassed }
  }, [messages, checkSet])

  return (
    <div className="admin-scoring-overview">
      <header className="aso-header">
        <div>
          <h1 className="aso-title">Scoring overview</h1>
        </div>
        <div className="aso-header-right">
          <ConnectionStatus status={status} error={error} />
        </div>
      </header>

      <p className="aso-live-hint">
        {status === 'connected'
          ? `${messages.length} result message${messages.length === 1 ? '' : 's'} in buffer`
          : status === 'connecting'
            ? 'Connecting to NATS…'
            : 'Not connected to NATS'}
        {checksLoadError && (
          <span className="aso-live-hint-err"> · Checks list: {checksLoadError}</span>
        )}
      </p>

      {columnChecks.length === 0 && !checksLoadError ? (
        <div className="aso-empty">Loading check columns…</div>
      ) : columnChecks.length === 0 ? (
        <div className="aso-empty">No checks returned from the API; the grid cannot be built.</div>
      ) : teamsSorted.length === 0 ? (
        <div className="aso-matrix-shell">
          <div className="aso-empty aso-empty--in-shell">
            No team results yet for configured checks. Waiting for NATS messages on{' '}
            <code className="aso-mono">results.&gt;</code>.
          </div>
        </div>
      ) : (
        <div className="aso-matrix-shell">
          <div className="aso-matrix-scroll">
            <table className="aso-matrix">
              <thead>
                <tr>
                  <th className="aso-matrix-corner" scope="col">
                    Team
                  </th>
                  {columnChecks.map(check => (
                    <th key={check} className="aso-matrix-colhead" title={check}>
                      {check}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {teamsSorted.map(team => (
                  <tr key={team}>
                    <th className="aso-matrix-rowhead" scope="row">
                      {team}
                    </th>
                    {columnChecks.map(check => {
                      const key = cellKey(team, check)
                      if (!latestPassedByCell.has(key)) {
                        return (
                          <td key={check} className="aso-matrix-cell aso-matrix-cell--empty">
                            <span className="aso-matrix-cell-inner">—</span>
                          </td>
                        )
                      }
                      const passed = latestPassedByCell.get(key) === true
                      return (
                        <td key={check} className="aso-matrix-cell">
                          <div
                            className={`aso-matrix-cell-inner aso-matrix-cell-inner--${passed ? 'pass' : 'fail'}`}
                            title={passed ? 'Latest run: passed' : 'Latest run: failed'}
                          >
                            <span className="aso-matrix-status">{passed ? 'Pass' : 'Fail'}</span>
                          </div>
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
