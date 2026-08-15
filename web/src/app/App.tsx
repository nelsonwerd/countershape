import { useCallback, useEffect, useMemo, useState } from "react";
import { StudioAPIError, StudioClient } from "../api/client";
import type { BenchResponse, DraftInput, MutationEnvelope, SessionResponse } from "../api/types";
import { DecisionBench } from "../features/decision/DecisionBench";
import { initialDraft } from "../features/decision/model";

type LoadState =
  | { kind: "loading" }
  | { kind: "error"; message: string; code: string }
  | { kind: "ready"; session: SessionResponse; bench: BenchResponse };

export function App() {
  const [client] = useState(() => {
    try {
      return StudioClient.fromLaunchLocation();
    } catch (error) {
      return error instanceof Error ? error : new Error("Studio launch failed.");
    }
  });
  const [state, setState] = useState<LoadState>({ kind: "loading" });
  const [pending, setPending] = useState(false);
  const [draft, setDraft] = useState<DraftInput | null>(null);
  const [notice, setNotice] = useState<{ code: string; message: string } | null>(null);

  const load = useCallback(async () => {
    if (client instanceof Error) {
      setState({ kind: "error", code: "STUDIO_LAUNCH_REFUSED", message: client.message });
      return;
    }
    try {
      const session = await client.session();
      const bench = await client.bench();
      setDraft((current) => current ?? bench.draft ?? initialDraft(bench));
      setState({ kind: "ready", session, bench });
    } catch (error) {
      setState(toErrorState(error));
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  const mutate = useCallback(
    async (operation: "visit" | "propose" | "reveal" | "revise" | "finalize", input: Omit<MutationEnvelope, "expected_revision">) => {
      if (client instanceof Error || state.kind !== "ready") return;
      setPending(true);
      setNotice(null);
      try {
        const bench = await client.mutate(operation, { ...input, expected_revision: state.bench.revision_digest });
        if (bench.draft !== undefined) setDraft(bench.draft);
        setState({ ...state, bench });
      } catch (error) {
        if (error instanceof StudioAPIError && error.code === "STUDIO_STALE_REVISION") {
          await load();
          setNotice({ code: error.code, message: error.message });
        } else if (error instanceof StudioAPIError) {
          setNotice({ code: error.code, message: error.message });
        } else {
          setState(toErrorState(error));
        }
      } finally {
        setPending(false);
      }
    },
    [client, load, state],
  );

  const progress = useMemo(() => {
    if (state.kind !== "ready") return 0;
    if (state.bench.result) return 3;
    if (state.bench.blind) return 2;
    if (state.bench.study_state === "EMPTY" || state.bench.study_state === "PREPARING") return 0;
    return 1;
  }, [state]);

  return (
    <div className="app-shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Countershape Studio home">
          <span className="brand-mark" aria-hidden="true"><i /><i /><i /></span>
          <span><strong>Countershape</strong><small>Evidence studio</small></span>
        </a>
        <div className="topbar-actions">
          <span className="security-chip"><span aria-hidden="true">●</span> Local session</span>
          <a href="#scope">Authority &amp; limits</a>
        </div>
      </header>
      <div className="workspace">
        <aside className="rail" aria-label="Workflow progress">
          <p className="eyebrow">Current workflow</p>
          <ol className="progress-list">
            {[
              ["Study", "Collect bounded evidence"],
              ["Compare", "Inspect exact differences"],
              ["Decide", "Record a local ruling"],
            ].map(([label, description], index) => (
              <li key={label} className={progress >= index + 1 ? "complete" : progress === index ? "current" : "future"}>
                <span className="step-index">{progress >= index + 1 ? "✓" : index + 1}</span>
                <span><strong>{label}</strong><small>{description}</small></span>
              </li>
            ))}
          </ol>
          <div className="rail-note">
            <span className="lock-icon" aria-hidden="true">⌾</span>
            <p><strong>Trusted-local, not sandboxed.</strong> Compared repository code runs with your user and host-network permissions.</p>
          </div>
        </aside>
        <main id="main-content">
          {state.kind === "loading" && <LoadingView />}
          {state.kind === "error" && <ErrorView code={state.code} message={state.message} retry={() => void load()} />}
          {state.kind === "ready" && (
            <DecisionBench
              bench={state.bench}
              draft={draft ?? initialDraft(state.bench)}
              setDraft={setDraft}
              pending={pending}
              notice={notice}
              mutate={mutate}
            />
          )}
        </main>
      </div>
    </div>
  );
}

function LoadingView() {
  return (
    <section className="center-state" aria-live="polite">
      <span className="spinner" aria-hidden="true" />
      <h1>Opening the local evidence boundary</h1>
      <p>Authenticating this tab and loading package-issued facts.</p>
    </section>
  );
}

function ErrorView({ code, message, retry }: { code: string; message: string; retry: () => void }) {
  return (
    <section className="center-state error-state" role="alert">
      <span className="state-symbol" aria-hidden="true">!</span>
      <p className="eyebrow">Request refused · {code}</p>
      <h1>The studio could not establish this view</h1>
      <p>{message}</p>
      <button className="button secondary" onClick={retry}>Retry exact read</button>
    </section>
  );
}

function toErrorState(error: unknown): LoadState {
  if (error instanceof StudioAPIError) {
    return { kind: "error", code: error.code, message: error.message };
  }
  return { kind: "error", code: "STUDIO_CLIENT_REFUSED", message: error instanceof Error ? error.message : "The local studio request failed." };
}
