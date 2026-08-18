<script lang="ts">
	import { m } from '../paraglide/messages.js';
</script>

<div class="card bg-base-100 shadow-xl">
	<div class="card-body">
		<h2 class="card-title text-3xl">{m.welcome_message()}</h2>
		<p class="text-base-content/70">{m.welcome_description()}</p>
		<div class="card-actions mt-6">
			<a href="/dashboard" class="btn btn-primary">{m.go_to_dashboard()}</a>
			<a href="/test" class="btn btn-secondary">{m.test_route()}</a>
		</div>
	</div>
</div>
