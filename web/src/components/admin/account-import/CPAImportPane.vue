<script setup lang="ts">
import type { FileImportModel } from './types'

const model = defineModel<FileImportModel>({
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

function onPickFiles(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files) return
  model.value.files = Array.from(input.files)
  input.value = ''
}

function removeFile(index: number) {
  model.value.files.splice(index, 1)
}
</script>

<template>
  <div class="pane">
    <el-form label-width="120px">
      <el-form-item label="CPA 文件">
        <div class="file-actions">
          <input accept=".json,.txt" :disabled="disabled" type="file" multiple @change="onPickFiles">
          <div class="pane-hint">支持直接上传 CPA JSON/TXT，或把内容粘贴到下方。</div>
        </div>
      </el-form-item>

      <el-form-item v-if="model.files.length" label="已选文件">
        <div class="file-list">
          <el-tag
            v-for="(file, index) in model.files"
            :key="`${file.name}-${index}`"
            closable
            :disable-transitions="true"
            @close="removeFile(index)"
          >
            {{ file.name }}
          </el-tag>
        </div>
      </el-form-item>

      <el-form-item label="文本内容">
        <el-input
          v-model="model.text"
          :autosize="{ minRows: 8, maxRows: 14 }"
          :disabled="disabled"
          type="textarea"
          placeholder="可直接粘贴 CPA JSON 内容"
        />
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.file-actions {
  width: 100%;
}

.file-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pane-hint {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
