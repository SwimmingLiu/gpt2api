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

function clearFiles() {
  model.value.files = []
}
</script>

<template>
  <div class="pane">
    <el-alert
      type="info"
      :closable="false"
      title="支持本地 sub2api JSON 文件，后续由父组件决定如何调用统一导入接口。"
    />

    <el-form label-width="120px" style="margin-top: 16px">
      <el-form-item label="sub2api 文件">
        <div class="file-actions">
          <input accept=".json" :disabled="disabled" type="file" multiple @change="onPickFiles">
          <div class="pane-hint">可上传一个或多个 JSON 文件。</div>
        </div>
      </el-form-item>

      <el-form-item v-if="model.files.length" label="已选文件">
        <div class="file-list">
          <el-tag
            v-for="(file, index) in model.files"
            :key="`${file.name}-${index}`"
            :disable-transitions="true"
          >
            {{ file.name }}
          </el-tag>
          <el-button link type="primary" :disabled="disabled" @click="clearFiles">清空</el-button>
        </div>
      </el-form-item>

      <el-form-item label="文本内容">
        <el-input
          v-model="model.text"
          :autosize="{ minRows: 8, maxRows: 14 }"
          :disabled="disabled"
          type="textarea"
          placeholder="也可以粘贴 sub2api JSON 文本"
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
  align-items: center;
}

.pane-hint {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
