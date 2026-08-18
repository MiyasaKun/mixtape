<script lang="ts">
	import { m } from '../../paraglide/messages.js';
</script>

<div class="card bg-base-100 shadow-xl">
	<div class="card-body">
		<h2 class="card-title text-3xl">{m.dashboard_title()}</h2>
		<p class="text-base-content/70">{m.dashboard_description()}</p>
		<div class="card-actions mt-6">
			<a href="/" class="link link-primary">{m.back_to_home()}</a>
		</div>
	</div>
</div>
