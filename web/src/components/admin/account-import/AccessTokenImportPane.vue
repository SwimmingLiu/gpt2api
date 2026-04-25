<script setup lang="ts">
import { computed } from 'vue'
import type { AccessTokenImportModel } from './types'

const model = defineModel<AccessTokenImportModel>({
  required: true,
})

withDefaults(
  defineProps<{
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const tokenCount = computed(() =>
  model.value.tokens_text
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean).length,
)
</script>

<template>
  <div class="pane">
    <el-form label-width="120px">
      <el-form-item label="导入模式">
        <el-radio-group v-model="model.mode" :disabled="disabled">
          <el-radio-button label="AT" value="at" />
          <el-radio-button label="RT" value="rt" />
          <el-radio-button label="ST" value="st" />
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Client ID">
        <el-input
          v-model="model.client_id"
          :disabled="disabled"
          placeholder="RT 模式必填，AT/ST 可选"
        />
      </el-form-item>

      <el-form-item label="Token 列表">
        <el-input
          v-model="model.tokens_text"
          :autosize="{ minRows: 8, maxRows: 14 }"
          :disabled="disabled"
          type="textarea"
          placeholder="每行一个 token"
        />
        <div class="pane-hint">当前共 {{ tokenCount }} 行。</div>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.pane-hint {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
