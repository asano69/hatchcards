import NavBar from "../../components/NavBar";
import Admin from "./Admin";
import Connections from "./Connections";

// Settings combines the Admin and Connections sections onto one page.
// Each section stays in its own file (Admin.jsx / Connections.jsx); this
// file just lays them out one after another — no tabs, no extra state.
export default function Settings() {
  return (
    <div class="mx-auto flex min-h-screen w-full max-w-2xl flex-col gap-12 bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <NavBar />
      <h1 class="font-serif text-4xl">Settings</h1>
    
      <Connections />
      <Admin />

    </div>
  );
}
