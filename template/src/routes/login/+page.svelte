<script lang="ts">
	import { m } from '../../paraglide/messages.js';
	import { signInWithGoogle } from '#lib/stores/auth.js';

	let busy = $state(false);
	let error = $state('');
	let errorEl = $state<HTMLDivElement>();

	async function handleSignIn() {
		busy = true;
		error = '';
		try {
			await signInWithGoogle();
		} catch {
			error = m.sign_in_error();
			busy = false;
			await Promise.resolve().then(() => errorEl?.focus());
		}
	}
</script>

<svelte:head><title>{m.login_title()} · {m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-md">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body items-center text-center">
			<h1 class="card-title text-3xl text-balance">{m.login_title()}</h1>
			<p class="text-base-content/70 text-pretty">{m.login_description()}</p>

			<button
				class="btn btn-primary mt-6 w-full gap-3"
				onclick={handleSignIn}
				disabled={busy}
				aria-busy={busy}
			>
				<svg
					class="h-5 w-5"
					viewBox="0 0 24 24"
					xmlns="http://www.w3.org/2000/svg"
					aria-hidden="true"
				>
					<path
						fill="#4285F4"
						d="M23.06 12.25c0-.85-.08-1.67-.22-2.45H12v4.64h6.2a5.3 5.3 0 0 1-2.3 3.48v2.89h3.72c2.18-2 3.44-4.96 3.44-8.56Z"
					/>
					<path
						fill="#34A853"
						d="M12 24c3.1 0 5.7-1.03 7.62-2.79l-3.72-2.89c-1.03.69-2.35 1.1-3.9 1.1-3 0-5.55-2.03-6.46-4.76H1.66v2.98A11.99 11.99 0 0 0 12 24Z"
					/>
					<path
						fill="#FBBC05"
						d="M5.54 14.66a7.2 7.2 0 0 1 0-4.61V7.07H1.66a12 12 0 0 0 0 10.57l3.88-2.98Z"
					/>
					<path
						fill="#EA4335"
						d="M12 4.75c1.69 0 3.2.58 4.4 1.72l3.3-3.3C17.7 1.19 15.1 0 12 0A11.99 11.99 0 0 0 1.66 7.07l3.88 2.98C6.45 7.32 9 5.29 12 4.75Z"
					/>
				</svg>
				{busy ? m.loading() : m.continue_with_google()}
			</button>

			{#if error}
				<div
					bind:this={errorEl}
					role="alert"
					aria-live="polite"
					tabindex="-1"
					class="mt-4 flex w-full items-start gap-2 rounded-xl border border-black/[0.08] bg-error/10 p-3 text-sm text-error focus:outline-none"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="mt-0.5 h-4 w-4 shrink-0"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
						/>
					</svg>
					<span>{error}</span>
				</div>
			{/if}
		</div>
	</div>
</div>
